package email

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mindflow/internal/email/storage"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

// Config defines encrypted SMTP delivery and sender identity.
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Mode     string `json:"mode"`
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	From     string `json:"from"`
	Name     string `json:"name"`
}

type mailJob struct {
	UserID    string `json:"userId,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	TokenHash string `json:"tokenHash"`
}

// Send delivers a message using the configured TLS mode.
func Send(ctx context.Context, config Config, toValue, subject, body string) error {
	if config.Mode != "tls" && config.Mode != "starttls" {
		return errTLSRequired
	}
	if strings.ContainsAny(toValue+subject+config.From+config.Name, "\r\n") {
		return errMailHeader
	}
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	dialer := &net.Dialer{Timeout: smtpConnectSeconds * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return wrap("mail_worker.go", err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(smtpWriteSeconds * time.Second))
	tlsConfig := &tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}
	if config.Mode == "tls" {
		secure := tls.Client(conn, tlsConfig)
		err = secure.HandshakeContext(ctx)
		if err != nil {
			return wrap("mail_worker.go", err)
		}
		conn = secure
	}
	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return wrap("mail_worker.go", err)
	}
	defer func() { _ = client.Close() }()
	if config.Mode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errSTARTTLSRequired
		}
		err = client.StartTLS(tlsConfig)
		if err != nil {
			return wrap("mail_worker.go", err)
		}
	}
	if config.Username != "" {
		err = client.Auth(smtp.PlainAuth("", config.Username, config.Password, config.Host))
		if err != nil {
			return wrap("mail_worker.go", err)
		}
	}
	err = client.Mail(config.From)
	if err != nil {
		return wrap("mail_worker.go", err)
	}
	err = client.Rcpt(toValue)
	if err != nil {
		return wrap("mail_worker.go", err)
	}
	writer, err := client.Data()
	if err != nil {
		return wrap("mail_worker.go", err)
	}
	message, err := FormatMessage(config, toValue, subject, body)
	if err != nil {
		return err
	}
	_, err = io.WriteString(writer, message)
	if err != nil {
		return wrap("mail_worker.go", err)
	}
	err = writer.Close()
	if err != nil {
		return wrap("mail_worker.go", err)
	}

	return wrap("mail_worker.go", client.Quit())
}

func (a *Service) runMail(ctx context.Context) {
	var resourceID, payload string
	var attempts int
	dueMailRow, err := a.repository.EmailDueMail(ctx, time.Now().Unix())
	resourceID = dueMailRow.ID
	payload = dueMailRow.Payload
	attempts = dueMailRow.Attempts
	if err != nil {
		return
	}
	var job mailJob
	if json.Unmarshal([]byte(payload), &job) != nil {
		_, _ = a.repository.EmailDeleteJob(ctx, resourceID)

		return
	}
	valid, err := a.canDeliver(ctx, job)
	if err != nil {
		return
	}
	if !valid {
		_, _ = a.repository.EmailDeleteJob(ctx, resourceID)

		return
	}
	s, err := a.configuration.DeliveryConfig(ctx)
	if err == nil {
		err = Send(ctx, s, job.To, job.Subject, job.Body)
	}
	if err == nil {
		_ = a.recordDelivery(ctx, resourceID, job)

		return
	}
	// Deliberately do not store server error text that could echo credentials or links.
	attempts++
	if attempts >= maxMailAttempts {
		_, _ = a.repository.EmailFailMail(ctx, storage.EmailFailMailParams{Attempts: attempts, ID: resourceID})

		return
	}
	delay := int64(retryBaseSeconds * (1 << min(attempts, maxMailAttempts)))
	_, _ = a.repository.EmailRetryMail(ctx, storage.EmailRetryMailParams{Attempts: attempts, RunAt: time.Now().Unix() + delay, ID: resourceID})
}

var errSTARTTLSRequired = errors.New("SMTP server does not support STARTTLS")

var (
	errTLSRequired = errors.New("SMTP TLS or STARTTLS is required")
	errMailHeader  = errors.New("invalid email header")
)

func (a *Service) canDeliver(ctx context.Context, job mailJob) (bool, error) {
	if strings.HasPrefix(job.Purpose, "release-") {
		return a.authorization.CanDeliverUpdate(ctx, job.UserID)
	}
	if strings.HasPrefix(job.Purpose, "inactivity-") {
		return a.authorization.CanDeliverInactivity(ctx, job.UserID, job.Purpose)
	}
	if job.Purpose == "welcome" {
		return a.authorization.CanDeliverNotice(ctx, job.UserID)
	}

	return a.authorization.CanDeliver(ctx, job.TokenHash)
}

func (a *Service) recordDelivery(ctx context.Context, resourceID string, job mailJob) error {
	transaction, err := a.repository.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = transaction.Rollback() }()
	queries := a.repository.Within(transaction)
	if strings.HasPrefix(job.Purpose, "inactivity-") {
		err = queries.EmailRecordDelivery(ctx, storage.EmailRecordDeliveryParams{DedupKey: job.UserID + ":" + job.Purpose, UserID: job.UserID, DeliveredAt: int(time.Now().Unix())})
		if err != nil {
			return err
		}
	}
	_, err = queries.EmailDeleteJob(ctx, resourceID)
	if err != nil {
		return err
	}

	return wrap("record mail delivery", transaction.Commit())
}
