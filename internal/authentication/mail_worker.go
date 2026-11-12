package authentication

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"mindflow/internal/authentication/dbquery"
	"mindflow/internal/fault"
	"mindflow/internal/identifier"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type mailJob struct {
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	TokenHash string `json:"tokenHash"`
}

func sendMail(ctx context.Context, config SMTPConfig, toValue, subject, body string) error {
	addr := net.JoinHostPort(config.Host, strconv.Itoa(config.Port))
	dialer := &net.Dialer{Timeout: smtpConnectSeconds * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fault.Wrap("mail_worker.go", err)
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(smtpWriteSeconds * time.Second))
	tlsConfig := &tls.Config{ServerName: config.Host, MinVersion: tls.VersionTLS12}
	if config.Mode == "tls" {
		secure := tls.Client(conn, tlsConfig)
		err = secure.HandshakeContext(ctx)
		if err != nil {
			return fault.Wrap("mail_worker.go", err)
		}
		conn = secure
	}
	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return fault.Wrap("mail_worker.go", err)
	}
	defer func() { _ = client.Close() }()
	if config.Mode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errSTARTTLSRequired
		}
		err = client.StartTLS(tlsConfig)
		if err != nil {
			return fault.Wrap("mail_worker.go", err)
		}
	}
	if config.Username != "" {
		err = client.Auth(smtp.PlainAuth("", config.Username, config.Password, config.Host))
		if err != nil {
			return fault.Wrap("mail_worker.go", err)
		}
	}
	err = client.Mail(config.From)
	if err != nil {
		return fault.Wrap("mail_worker.go", err)
	}
	err = client.Rcpt(toValue)
	if err != nil {
		return fault.Wrap("mail_worker.go", err)
	}
	writer, err := client.Data()
	if err != nil {
		return fault.Wrap("mail_worker.go", err)
	}
	from := (&mail.Address{Name: config.Name, Address: config.From}).String()
	message := "From: " + from + "\r\nTo: " + toValue + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + strings.ReplaceAll(body, "\n", "\r\n")
	_, err = io.WriteString(writer, message)
	if err != nil {
		return fault.Wrap("mail_worker.go", err)
	}
	err = writer.Close()
	if err != nil {
		return fault.Wrap("mail_worker.go", err)
	}

	return fault.Wrap("mail_worker.go", client.Quit())
}

// RunWorkers processes due jobs until the context is canceled.
func (a *Service) RunWorkers(ctx context.Context) {
	tick := time.NewTicker(workerIntervalSeconds * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			a.runMail(ctx)
			a.cleanup(ctx)
		}
	}
}

func (a *Service) runMail(ctx context.Context) {
	var resourceID, payload string
	var attempts int
	dueMailRow, err := dbquery.New(a.db).DueMail(ctx, identifier.Now())
	resourceID = dueMailRow.ID
	payload = dueMailRow.Payload
	attempts = dueMailRow.Attempts
	if err != nil {
		return
	}
	var job mailJob
	if json.Unmarshal([]byte(payload), &job) != nil {
		_, _ = dbquery.New(a.db).DeleteJob(ctx, resourceID)

		return
	}
	var valid int
	countValidTokensRow, err := dbquery.New(a.db).CountValidTokens(ctx, dbquery.CountValidTokensParams{TokenHash: job.TokenHash, ExpiresAt: identifier.Now()})
	valid = int(countValidTokensRow)
	if err != nil {
		return
	}
	if valid == 0 {
		_, _ = dbquery.New(a.db).DeleteJob(ctx, resourceID)

		return
	}
	s, err := a.settings(ctx)
	if err == nil {
		err = sendMail(ctx, s.SMTP, job.To, job.Subject, job.Body)
	}
	if err == nil {
		_, _ = dbquery.New(a.db).DeleteJob(ctx, resourceID)

		return
	}
	// Deliberately do not store server error text that could echo credentials or links.
	attempts++
	if attempts >= maxMailAttempts {
		_, _ = dbquery.New(a.db).FailMail(ctx, dbquery.FailMailParams{Attempts: attempts, ID: resourceID})

		return
	}
	delay := int64(retryBaseSeconds * (1 << min(attempts, maxMailAttempts)))
	_, _ = dbquery.New(a.db).RetryMail(ctx, dbquery.RetryMailParams{Attempts: attempts, RunAt: identifier.Now() + delay, ID: resourceID})
}

// SendMail delivers a message using the configured TLS mode.
func (a *Service) SendMail(ctx context.Context, c SMTPConfig, to, subject, body string) error {
	return sendMail(ctx, c, to, subject, body)
}

var errSTARTTLSRequired = errors.New("SMTP server does not support STARTTLS")
