package providers

import (
	"crypto/tls"
	"fmt"
	"net"

	"strings"

	"github.com/emersion/go-sasl"
	esmtp "github.com/emersion/go-smtp"
	"github.com/foolin/goview"
	"github.com/labstack/echo/v4"
)

type SmptClient struct {
	Server   string
	Login    string
	Password string
}

func NewSmptClient(server, login, password string) *SmptClient {
	return &SmptClient{
		Server:   server,
		Login:    login,
		Password: password,
	}
}

func (s *SmptClient) SendMail(dest []string, msg string) error {
	host, _, _ := net.SplitHostPort(s.Server)
	c, err := esmtp.DialTLS(s.Server, &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	})
	if err != nil {
		return err
	}

	if err := c.Auth(sasl.NewPlainClient("", s.Login, s.Password)); err != nil {
		return err
	}

	if err := c.Mail(s.Login); err != nil {
		return err
	}

	for _, r := range dest {
		if err := c.Rcpt(r); err != nil {
			return err
		}
	}

	wc, err := c.Data()
	if err != nil {
		return err
	}

	if _, err = fmt.Fprint(wc, msg); err != nil {
		return err
	}

	if err = wc.Close(); err != nil {
		return err
	}

	if err = c.Quit(); err != nil {
		return err
	}

	return nil
}

type EmailSender struct {
	t *goview.ViewEngine
	c *SmptClient
}

func NewEmailSender(client *SmptClient, templates string) *EmailSender {
	return &EmailSender{
		t: goview.New(goview.Config{
			Root:      "templates/emails",
			Extension: ".tmpl",
			Master:    "layouts/base",
			Partials:  []string{"assets/headers", "assets/style"},
		}),
		c: client,
	}
}

func (es *EmailSender) Send(dest []string, template string, data echo.Map) error {
	msg := &strings.Builder{}
	if err := es.t.RenderWriter(msg, template, data); err != nil {
		return err
	}

	return es.c.SendMail(dest, msg.String())
}
