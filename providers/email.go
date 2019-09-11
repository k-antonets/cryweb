package providers

import (
	"crypto/tls"
	"fmt"
	"html/template"
	"net"
	"net/smtp"

	"strings"

	"github.com/foolin/goview"
	"github.com/jordan-wright/email"
	"github.com/labstack/echo/v4"
	"github.com/vanng822/go-premailer/premailer"
)

type SmptClient struct {
	Server   string
	Name     string
	Login    string
	Password string
}

func NewSmptClient(server, login, name, password string) *SmptClient {
	return &SmptClient{
		Server:   server,
		Login:    login,
		Password: password,
	}
}

func (s *SmptClient) SendMail(dest []string, subject, msg string) error {
	e := email.NewEmail()
	host, _, _ := net.SplitHostPort(s.Server)
	e.From = fmt.Sprintf("%s <%s>", s.Name, s.Login)
	e.To = dest
	e.Subject = subject
	e.HTML = []byte(msg)
	fmt.Println(e.From)
	return e.SendWithTLS(s.Server, smtp.PlainAuth("", s.Login, s.Password, host), &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	})
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
			Partials:  []string{"assets/headers", "assets/style", "assets/recipient"},
			Funcs:     template.FuncMap{"join": strings.Join},
		}),
		c: client,
	}
}

func (es *EmailSender) Send(dest []string, subject, template string, data echo.Map) error {
	msg := &strings.Builder{}
	data["sender"] = es.c.Login
	data["recipient"] = dest
	data["subject"] = subject
	if err := es.t.RenderWriter(msg, template, data); err != nil {
		return err
	}

	msgStr := strings.ReplaceAll(msg.String(), "\n", "\r\n")
	prem, err := premailer.NewPremailerFromString(msgStr, premailer.NewOptions())
	if err != nil {
		return err
	}
	msgStr, err = prem.Transform()
	if err != nil {
		return err
	}

	return es.c.SendMail(dest, subject, msgStr)
}
