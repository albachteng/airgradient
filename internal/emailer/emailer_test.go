package emailer_test

import (
	"net/smtp"
	"strings"
	"testing"

	"github.com/albachteng/airgradient/internal/emailer"
)

type sentMail struct {
	addr string
	from string
	to   []string
	msg  []byte
}

// captureSendMail returns a sendMail-compatible func that records what was sent.
func captureSendMail(out *sentMail) func(string, smtp.Auth, string, []string, []byte) error {
	return func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
		out.addr = addr
		out.from = from
		out.to = to
		out.msg = msg
		return nil
	}
}

func TestSendCallsSMTP(t *testing.T) {
	var sent sentMail
	cfg := emailer.Config{
		SMTPHost: "smtp.gmail.com",
		SMTPPort: 587,
		Username: "sender@gmail.com",
		Password: "app-password",
		From:     "sender@gmail.com",
		To:       []string{"recipient@gmail.com"},
	}
	e := emailer.NewWithSender(cfg, captureSendMail(&sent))
	if err := e.Send("Test Subject", "<p>Hello</p>"); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if sent.addr != "smtp.gmail.com:587" {
		t.Errorf("addr: got %q, want %q", sent.addr, "smtp.gmail.com:587")
	}
	if sent.from != "sender@gmail.com" {
		t.Errorf("from: got %q, want %q", sent.from, "sender@gmail.com")
	}
	if len(sent.to) != 1 || sent.to[0] != "recipient@gmail.com" {
		t.Errorf("to: got %v, want [recipient@gmail.com]", sent.to)
	}
}

func TestSendMessageContainsHeaders(t *testing.T) {
	var sent sentMail
	cfg := emailer.Config{
		SMTPHost: "smtp.gmail.com",
		SMTPPort: 587,
		Username: "sender@gmail.com",
		Password: "secret",
		From:     "sender@gmail.com",
		To:       []string{"you@gmail.com"},
	}
	e := emailer.NewWithSender(cfg, captureSendMail(&sent))
	e.Send("Weekly Air Quality Report", "<p>Data</p>")

	msg := string(sent.msg)
	if !strings.Contains(msg, "Subject: Weekly Air Quality Report") {
		t.Errorf("message missing Subject header:\n%s", msg)
	}
	if !strings.Contains(msg, "From: sender@gmail.com") {
		t.Errorf("message missing From header:\n%s", msg)
	}
	if !strings.Contains(msg, "To: you@gmail.com") {
		t.Errorf("message missing To header:\n%s", msg)
	}
	if !strings.Contains(msg, `Content-Type: text/html; charset="utf-8"`) {
		t.Errorf("message missing Content-Type header:\n%s", msg)
	}
}

func TestSendMessageContainsBody(t *testing.T) {
	var sent sentMail
	cfg := emailer.Config{
		SMTPHost: "smtp.gmail.com",
		SMTPPort: 587,
		Username: "sender@gmail.com",
		Password: "secret",
		From:     "sender@gmail.com",
		To:       []string{"you@gmail.com"},
	}
	e := emailer.NewWithSender(cfg, captureSendMail(&sent))
	e.Send("Subject", "<h1>Air Quality</h1><p>PM2.5: 5.3</p>")

	if !strings.Contains(string(sent.msg), "<h1>Air Quality</h1>") {
		t.Errorf("message body missing HTML content:\n%s", string(sent.msg))
	}
}

func TestSendMultipleRecipients(t *testing.T) {
	var sent sentMail
	cfg := emailer.Config{
		SMTPHost: "smtp.gmail.com",
		SMTPPort: 587,
		Username: "sender@gmail.com",
		Password: "secret",
		From:     "sender@gmail.com",
		To:       []string{"alice@example.com", "bob@example.com"},
	}
	e := emailer.NewWithSender(cfg, captureSendMail(&sent))
	e.Send("Subject", "<p>Body</p>")

	if len(sent.to) != 2 {
		t.Errorf("expected 2 recipients, got %d: %v", len(sent.to), sent.to)
	}
}
