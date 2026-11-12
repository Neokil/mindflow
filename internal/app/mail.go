package app

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type mailJob struct{ To, Subject, Body, TokenHash string }

func sendMail(ctx context.Context, c SMTPConfig, to, subject, body string) error {
	addr := net.JoinHostPort(c.Host, fmt.Sprint(c.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	tlsConfig := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
	if c.Mode == "tls" {
		secure := tls.Client(conn, tlsConfig)
		if err = secure.HandshakeContext(ctx); err != nil {
			return err
		}
		conn = secure
	}
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if c.Mode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server does not support STARTTLS")
		}
		if err = client.StartTLS(tlsConfig); err != nil {
			return err
		}
	}
	if c.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); err != nil {
			return err
		}
	}
	if err = client.Mail(c.From); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	from := (&mail.Address{Name: c.Name, Address: c.From}).String()
	message := "From: " + from + "\r\nTo: " + to + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + strings.ReplaceAll(body, "\n", "\r\n")
	if _, err = io.WriteString(writer, message); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
func (a *App) RunWorkers(ctx context.Context) {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			a.runMail(ctx)
		}
	}
}
func (a *App) runMail(ctx context.Context) {
	var id, payload string
	var attempts int
	if err := a.DB.QueryRowContext(ctx, "SELECT id,payload,attempts FROM jobs WHERE kind='mail' AND run_at<=? AND attempts<8 ORDER BY run_at LIMIT 1", now()).Scan(&id, &payload, &attempts); err != nil {
		return
	}
	var job mailJob
	if json.Unmarshal([]byte(payload), &job) != nil {
		_, _ = a.DB.ExecContext(ctx, "DELETE FROM jobs WHERE id=?", id)
		return
	}
	var valid int
	if err := a.DB.QueryRowContext(ctx, "SELECT count(*) FROM account_tokens t JOIN users u ON u.id=t.user_id WHERE t.token_hash=? AND t.expires_at>? AND u.status IN ('active','invited')", job.TokenHash, now()).Scan(&valid); err != nil {
		return
	}
	if valid == 0 {
		_, _ = a.DB.ExecContext(ctx, "DELETE FROM jobs WHERE id=?", id)
		return
	}
	s, err := a.settings()
	if err == nil {
		err = sendMail(ctx, s.SMTP, job.To, job.Subject, job.Body)
	}
	if err == nil {
		_, _ = a.DB.ExecContext(ctx, "DELETE FROM jobs WHERE id=?", id)
		return
	}
	// Deliberately do not store server error text that could echo credentials or links.
	attempts++
	if attempts >= 8 {
		_, _ = a.DB.ExecContext(ctx, "UPDATE jobs SET attempts=?,payload='{}',last_error='Email delivery failed. Check SMTP settings and resend.' WHERE id=?", attempts, id)
		return
	}
	delay := int64(15 * (1 << min(attempts, 8)))
	_, _ = a.DB.ExecContext(ctx, "UPDATE jobs SET attempts=?,run_at=?,last_error='Email delivery failed; retry scheduled.' WHERE id=?", attempts, now()+delay, id)
}
