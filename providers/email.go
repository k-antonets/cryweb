package providers

import (
	"html/template"
	"net/smtp"
	"strings"

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
	t *template.Template
	c *SmptClient
}

func NewEmailSender(client *SmptClient, templates string) *EmailSender {
	return &EmailSender{
		t: template.Must(template.ParseGlob(templates)),
		c: client,
	}
}

func (es *EmailSender) Send(dest []string, template string, data echo.Map) error {
	msg := &strings.Builder{}
	if err := es.t.ExecuteTemplate(msg, "headers", data); err != nil {
		return err
	}
	if err := es.t.ExecuteTemplate(msg, template, data); err != nil {
		return err
	}

	return es.c.SendMail(dest, msg.String())
}
