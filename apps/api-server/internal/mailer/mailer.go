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
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host        string
	Port        int
	User        string
	Pass        string
	From        string
	FromName    string
	Encryption  string
	AuthEnabled bool
	SkipVerify  bool
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
	return m.send(to, subject, body, false)
}

// SendText sends a plain UTF-8 text email for admin notices.
func (m *Mailer) SendText(to, subject, body string) error {
	return m.send(to, subject, body, false)
}

// SendHTML sends a UTF-8 HTML email.
func (m *Mailer) SendHTML(to, subject, body string) error {
	return m.send(to, subject, body, true)
}

func (m *Mailer) send(to, subject, body string, html bool) error {
	if !m.Enabled() {
		log.Printf("[mailer DISABLED] to=%s subject=%q body=%q", to, subject, body)
		return nil
	}

	msg := m.buildMessage(to, subject, body, html)
	addr := net.JoinHostPort(m.cfg.Host, strconv.Itoa(m.cfg.Port))
	if strings.EqualFold(m.cfg.Encryption, "ssl") || strings.EqualFold(m.cfg.Encryption, "tls") || m.cfg.Port == 465 {
		return m.sendImplicitTLS(addr, to, msg)
	}
	return m.sendSMTP(addr, to, msg)
}

func (m *Mailer) buildMessage(to, subject, body string, html bool) []byte {
	from := m.cfg.From
	if strings.TrimSpace(m.cfg.FromName) != "" {
		from = mime.QEncoding.Encode("UTF-8", strings.TrimSpace(m.cfg.FromName)) + " <" + m.cfg.From + ">"
	}
	contentType := "text/plain; charset=UTF-8"
	if html {
		contentType = "text/html; charset=UTF-8"
	}
	headers := map[string]string{
		"From":         from,
		"To":           to,
		"Subject":      mime.QEncoding.Encode("UTF-8", subject),
		"MIME-Version": "1.0",
		"Content-Type": contentType,
	}
	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(k + ": " + v + "\r\n")
	}
	msg.WriteString("\r\n")
	msg.WriteString(body)
	return []byte(msg.String())
}

func (m *Mailer) sendSMTP(addr, to string, msg []byte) error {
	var auth smtp.Auth
	if m.cfg.AuthEnabled && m.cfg.User != "" {
		auth = smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)
	}
	if strings.EqualFold(m.cfg.Encryption, "none") || m.cfg.Encryption == "" {
		return smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg)
	}
	c, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer c.Quit()
	if ok, _ := c.Extension("STARTTLS"); !ok {
		return fmt.Errorf("smtp starttls not supported by server")
	}
	if err := c.StartTLS(&tls.Config{ServerName: m.cfg.Host, InsecureSkipVerify: m.cfg.SkipVerify}); err != nil {
		return fmt.Errorf("smtp starttls: %w", err)
	}
	if auth != nil {
		if err := c.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
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

func (m *Mailer) sendImplicitTLS(addr, to string, msg []byte) error {
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 15 * time.Second}, "tcp", addr,
		&tls.Config{ServerName: m.cfg.Host, InsecureSkipVerify: m.cfg.SkipVerify})
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()
	c, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Quit()
	if m.cfg.AuthEnabled && m.cfg.User != "" {
		if err := c.Auth(smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
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
