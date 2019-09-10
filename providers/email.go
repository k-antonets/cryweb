package providers

import (
	"net/smtp"
	"strings"

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
	m := "From: " + s.Login + "\n" +
		"To: " + strings.Join(dest, ",") + "\n"
	return smtp.SendMail(s.Server,
		smtp.PlainAuth("", s.Login, s.Password, s.Server),
		s.Login, dest, []byte(m))
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
