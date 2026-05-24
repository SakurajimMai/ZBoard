// Package mailer sends transactional emails via SMTP.
//
// Configure via env:
//
//	ZBOARD_SMTP_HOST, ZBOARD_SMTP_PORT, ZBOARD_SMTP_USER, ZBOARD_SMTP_PASS,
//	ZBOARD_SMTP_FROM (sender email)
//
// When SMTP is not configured, sends are logged and treated as no-ops so
// development environments don't need a real mail server.
package mailer

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

type Mailer struct {
	cfg Config
}

func New(cfg Config) *Mailer { return &Mailer{cfg: cfg} }

// Enabled returns true when SMTP is configured. When false, Send is a no-op.
func (m *Mailer) Enabled() bool { return m.cfg.Host != "" && m.cfg.From != "" }

// SendCode sends a verification code email. Subject and body are templated.
func (m *Mailer) SendCode(to, code, purpose string) error {
	subject := "Zboard 验证码"
	switch purpose {
	case "register":
		subject = "Zboard 注册验证码"
	case "reset_password":
		subject = "Zboard 密码重置验证码"
	}
	body := fmt.Sprintf(
		"您好，\n\n您的 Zboard 验证码是：%s\n\n验证码 10 分钟内有效。如非本人操作，请忽略此邮件。\n\n— Zboard",
		code)
	return m.send(to, subject, body)
}

// SendText sends a plain UTF-8 text email for admin notices.
func (m *Mailer) SendText(to, subject, body string) error {
	return m.send(to, subject, body)
}

func (m *Mailer) send(to, subject, body string) error {
	if !m.Enabled() {
		log.Printf("[mailer DISABLED] to=%s subject=%q body=%q", to, subject, body)
		return nil
	}

	headers := map[string]string{
		"From":         m.cfg.From,
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/plain; charset=UTF-8",
	}
	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(k + ": " + v + "\r\n")
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)

	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))

	// SMTPS (465) uses implicit TLS; STARTTLS (587) uses explicit TLS.
	if m.cfg.Port == 465 {
		return m.sendImplicitTLS(addr, to, []byte(msg.String()))
	}
	auth := smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)
	return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, []byte(msg.String()))
}

func (m *Mailer) sendImplicitTLS(addr, to string, msg []byte) error {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr,
		&tls.Config{ServerName: m.cfg.Host})
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Quit()
	if err := c.Auth(smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := c.Mail(m.cfg.From); err != nil {
		return err
	}
	if err := c.Rcpt(to); err != nil {
		return err
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	return w.Close()
}
