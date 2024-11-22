package smtp

import (
	"github.com/go-gomail/gomail"
	"github.com/zerodha/logf"
)

type Provider string

const (
	GMAIL      Provider = "gmail"
	MAILSENDER          = "mailersend"
)

type Smtp interface {
	Provider() Provider
	SendEmail(from string, to string, subject string, contentType, body string, attachments []File) error
	From() string
}

type smtp struct {
	log       logf.Logger
	provider  Provider
	host      string
	port      int
	username  string
	password  string
	fromEmail string
	fromName  string
}

func New(log logf.Logger, provider Provider, host string, port int, username, password, fromEmail, fromName string) Smtp {
	return &smtp{
		log:       log,
		provider:  provider,
		host:      host,
		port:      port,
		username:  username,
		password:  password,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

func (s *smtp) Provider() Provider {
	return s.provider
}

func (s *smtp) From() string {
	return s.fromEmail
}

func (s *smtp) SendEmail(from string, to string, subject string, contentType, body string, attachments []File) error {
	// To:      []string{"test@example.com"},
	// From:    "Jordan Wright <test@gmail.com>",
	m := gomail.NewMessage()
	if from == "" {
		from = m.FormatAddress(s.fromEmail, s.fromName)
	}
	if to == "" {
		to = m.FormatAddress(s.fromEmail, s.fromName)
	}
	if contentType == "" {
		contentType = "plain/text" // text/html
	}
	m.SetBody(contentType, body)
	m.SetHeaders(map[string][]string{
		"From":    {from},
		"To":      {to},
		"Subject": {subject},
	})
	for _, att := range attachments {
		m.Attach(att.Path, gomail.Rename(att.Name))
	}
	d := gomail.NewPlainDialer(s.host, s.port, s.username, s.password)
	return d.DialAndSend(m)
}
