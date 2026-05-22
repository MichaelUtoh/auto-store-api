package email

import (
	"fmt"
	"net/smtp"
	"strings"

	"auto-store-api/internal/config"

	"auto-store-api/pkg/logger"

	"go.uber.org/zap"
)

// Sender delivers plain-text or HTML emails.
type Sender interface {
	Send(to, subject, body string) error
}

type smtpSender struct {
	cfg config.EmailConfig
}

func NewSender(cfg config.EmailConfig) Sender {
	if cfg.Host == "" {
		return &logSender{}
	}
	return &smtpSender{cfg: cfg}
}

func (s *smtpSender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	auth := smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)
	msg := strings.Join([]string{
		"From: " + s.cfg.From,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	return smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg))
}

// logSender logs emails when SMTP is not configured (local dev).
type logSender struct{}

func (s *logSender) Send(to, subject, body string) error {
	logger.Log.Info("email (SMTP not configured)",
		zap.String("to", to),
		zap.String("subject", subject),
		zap.String("body", body),
	)
	return nil
}
