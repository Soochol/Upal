package notify

import (
	"context"
	"fmt"
	"net/smtp"

	"github.com/soochol/upal/internal/upal"
)

// SMTPSender sends messages via SMTP email.
type SMTPSender struct{}

func (s *SMTPSender) Type() upal.ConnectionType { return upal.ConnTypeSMTP }

func (s *SMTPSender) Send(ctx context.Context, conn *upal.Connection, message string) error {
	to, _ := conn.Extras["to"].(string)
	if to == "" {
		return fmt.Errorf("smtp connection %q missing 'to' in extras", conn.ID)
	}

	from := conn.Login
	if from == "" {
		return fmt.Errorf("smtp connection %q missing login (from address)", conn.ID)
	}

	subject, _ := conn.Extras["subject"].(string)
	if subject == "" {
		subject = "Upal Notification"
	}

	host := conn.Host
	port := conn.Port
	if port == 0 {
		port = 587
	}
	addr := fmt.Sprintf("%s:%d", host, port)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, to, subject, message)

	var auth smtp.Auth
	if conn.Password != "" {
		auth = smtp.PlainAuth("", from, conn.Password, host)
	}

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}
