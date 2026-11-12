package email

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mindflow/internal/email/storage"
	"net"
	"net/mail"
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
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	TokenHash string `json:"tokenHash"`
}

// Send delivers a message using the configured TLS mode.
func Send(ctx context.Context, config Config, toValue, subject, body string) error {
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
	from := (&mail.Address{Name: config.Name, Address: config.From}).String()
	message := "From: " + from + "\r\nTo: " + toValue + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + strings.ReplaceAll(body, "\n", "\r\n")
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
	dueMailRow, err := a.repository.DueMail(ctx, time.Now().Unix())
	resourceID = dueMailRow.ID
	payload = dueMailRow.Payload
	attempts = dueMailRow.Attempts
	if err != nil {
		return
	}
	var job mailJob
	if json.Unmarshal([]byte(payload), &job) != nil {
		_, _ = a.repository.DeleteJob(ctx, resourceID)

		return
	}
	valid, err := a.authorization.CanDeliver(ctx, job.TokenHash)
	if err != nil {
		return
	}
	if !valid {
		_, _ = a.repository.DeleteJob(ctx, resourceID)

		return
	}
	s, err := a.configuration.DeliveryConfig(ctx)
	if err == nil {
		err = Send(ctx, s, job.To, job.Subject, job.Body)
	}
	if err == nil {
		_, _ = a.repository.DeleteJob(ctx, resourceID)

		return
	}
	// Deliberately do not store server error text that could echo credentials or links.
	attempts++
	if attempts >= maxMailAttempts {
		_, _ = a.repository.FailMail(ctx, storage.FailMailParams{Attempts: attempts, ID: resourceID})

		return
	}
	delay := int64(retryBaseSeconds * (1 << min(attempts, maxMailAttempts)))
	_, _ = a.repository.RetryMail(ctx, storage.RetryMailParams{Attempts: attempts, RunAt: time.Now().Unix() + delay, ID: resourceID})
}

var errSTARTTLSRequired = errors.New("SMTP server does not support STARTTLS")
