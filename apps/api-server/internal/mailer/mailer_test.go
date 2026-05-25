package mailer

import (
	"bytes"
	"testing"
)

func TestBuildMessageUsesDisplayNameAndHTMLContentType(t *testing.T) {
	m := New(Config{
		Host:     "smtp.example.com",
		Port:     587,
		From:     "noreply@example.com",
		FromName: "Zboard",
	})
	msg := m.buildMessage("user@example.com", "测试主题", "<p>你好</p>", true)

	if !bytes.Contains(msg, []byte("From: Zboard <noreply@example.com>")) {
		t.Fatalf("missing display from header:\n%s", string(msg))
	}
	if bytes.Contains(msg, []byte("Subject: 测试主题")) {
		t.Fatalf("non-ascii subject should be MIME encoded:\n%s", string(msg))
	}
	if !bytes.Contains(msg, []byte("Content-Type: text/html; charset=UTF-8")) {
		t.Fatalf("missing html content type:\n%s", string(msg))
	}
	if !bytes.Contains(msg, []byte("<p>你好</p>")) {
		t.Fatalf("missing body:\n%s", string(msg))
	}
}

func TestBuildMessageFallsBackToPlainText(t *testing.T) {
	m := New(Config{Host: "smtp.example.com", Port: 587, From: "noreply@example.com"})
	msg := m.buildMessage("user@example.com", "测试主题", "正文", false)

	if !bytes.Contains(msg, []byte("Content-Type: text/plain; charset=UTF-8")) {
		t.Fatalf("missing text content type:\n%s", string(msg))
	}
}
