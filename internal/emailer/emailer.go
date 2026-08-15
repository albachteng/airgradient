package emailer

import (
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       []string
}

type Emailer struct {
	cfg      Config
	sendMail func(addr string, auth smtp.Auth, from string, to []string, msg []byte) error
}

func New(cfg Config) *Emailer {
	return NewWithSender(cfg, smtp.SendMail)
}

func NewWithSender(cfg Config, sender func(string, smtp.Auth, string, []string, []byte) error) *Emailer {
	return &Emailer{cfg: cfg, sendMail: sender}
}

func (e *Emailer) Send(subject, htmlBody string) error {
	addr := fmt.Sprintf("%s:%d", e.cfg.SMTPHost, e.cfg.SMTPPort)
	auth := smtp.PlainAuth("", e.cfg.Username, e.cfg.Password, e.cfg.SMTPHost)

	header := strings.Join([]string{
		"From: " + e.cfg.From,
		"To: " + strings.Join(e.cfg.To, ", "),
		"Subject: " + subject,
		"MIME-Version: 1.0",
		`Content-Type: text/html; charset="utf-8"`,
		"",
		"",
	}, "\r\n")

	msg := []byte(header + htmlBody)
	return e.sendMail(addr, auth, e.cfg.From, e.cfg.To, msg)
}
