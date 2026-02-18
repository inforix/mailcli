package smtp

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"time"

	"mailcli/internal/config"
)

func Send(cfg config.Config, from string, recipients []string, msg []byte) error {
	if len(recipients) == 0 {
		return fmt.Errorf("no recipients provided")
	}

	addr := fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port)
	host := cfg.SMTP.Host

	var c *smtp.Client
	var err error

	if cfg.SMTP.TLS {
		tlsConfig := &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: cfg.SMTP.InsecureSkipVerify,
		}
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 30 * time.Second}, "tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		c, err = smtp.NewClient(conn, host)
		if err != nil {
			return err
		}
	} else {
		c, err = smtp.Dial(addr)
		if err != nil {
			return err
		}
		if cfg.SMTP.StartTLS {
			tlsConfig := &tls.Config{
				ServerName:         host,
				InsecureSkipVerify: cfg.SMTP.InsecureSkipVerify,
			}
			if err := c.StartTLS(tlsConfig); err != nil {
				_ = c.Quit()
				return err
			}
		}
	}
	defer c.Quit()

	auth := smtp.PlainAuth("", cfg.Auth.Username, cfg.Auth.Password, host)
	if err := c.Auth(auth); err != nil {
		return err
	}

	if err := c.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range recipients {
		if err := c.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}

	return c.Quit()
}
