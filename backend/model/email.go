package model

import (
	"time"
)

type EmailType string

const (
	SMS_SENDING_FAILED EmailType = "sms-sending-failed"
	SMS_RECEIVED       EmailType = "sms-received"
)

func NewEmail(typ EmailType, from, to, subject, bodyHtml string) *Email {
	return &Email{
		Type:     typ,
		From:     from,
		To:       to,
		Subject:  subject,
		BodyHtml: bodyHtml,
		Created:  time.Now(),
	}
}

type Email struct {
	ID        uint       `json:"id" db:"id"`
	Type      EmailType  `json:"email_type" db:"email_type"`
	From      string     `json:"email_from" db:"email_from"`
	To        string     `json:"email_to" db:"email_to"`
	Subject   string     `json:"subject" db:"subject"`
	BodyHtml  string     `json:"body_html" db:"body_html"`
	Created   time.Time  `json:"created" db:"created"`
	Sent      *time.Time `json:"sent" db:"sent"`
	Retries   *int       `json:"retries" db:"retries"`
	LastError *string    `json:"last_error" db:"last_error"`
}

func (u *Email) TableName() string {
	return "email"
}

type EmailTemplate struct {
	Code    string `json:"code" db:"code"`
	Subject string `json:"subject" db:"subject"`
	Body    string `json:"body" db:"body"`
}

func (u *EmailTemplate) TableName() string {
	return "emailTemplates"
}
