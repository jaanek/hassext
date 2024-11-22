package mailer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/jaanek/hassext/html"
	"github.com/jaanek/hassext/i18n"
	"github.com/jaanek/hassext/model"
	"github.com/jaanek/hassext/smtp"
	"github.com/jaanek/hassext/sqldb"
	"github.com/jaanek/hassext/sqlite"

	"github.com/zerodha/logf"
)

type TemplateCode string

const (
	EMAIL_CONTACT TemplateCode = "contact"
)

type Mailer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	GetEmailTemplate(code TemplateCode, lang string) (*model.EmailTemplate, error)
	StoreEmail(e *model.Email) error
	SendUnsentEmails() error
}

type CheckEmail struct{}

type mailer struct {
	prefix       string
	log          logf.Logger
	done         chan struct{}
	doneOnce     sync.Once
	triggerCheck chan CheckEmail
	sqliteDir    string
	smtps        map[smtp.Provider]smtp.Smtp
}

func New(log logf.Logger, sqliteDir string, smtps ...smtp.Smtp) Mailer {
	var smtpMap = map[smtp.Provider]smtp.Smtp{}
	for _, smtp := range smtps {
		smtpMap[smtp.Provider()] = smtp
	}
	return &mailer{
		prefix:       "[mailer] ",
		log:          log,
		done:         make(chan struct{}),
		triggerCheck: make(chan CheckEmail), // unbuffered because one at a time send to gmail & sequential poll from emails table
		sqliteDir:    sqliteDir,
		smtps:        smtpMap,
	}
}

func (t *mailer) SendUnsentEmails() error {
	t.triggerCheck <- CheckEmail{}
	return nil
}

func (t *mailer) Start(ctx context.Context) (err error) {
	t.log.Info(t.prefix + "Starting Mailer")

	// start polling loop or react to triggering event
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-t.triggerCheck:
				err := t.sendUnsentEmails()
				if err != nil {
					t.log.Error(t.prefix+"error while sending email", "error", err)
				}
			case <-ticker.C:
				err := t.sendUnsentEmails()
				if err != nil {
					t.log.Error(t.prefix+"error while sending email", "error", err)
				}
			case <-t.done:
				t.log.Info(t.prefix + "Exiting mailer ...")
				return
			}
		}
	}()
	return nil
}

func (t *mailer) Stop(ctx context.Context) error {
	t.log.Info(t.prefix + "Stopping Mailer")
	t.doneOnce.Do(func() {
		close(t.done)
	})
	return nil
}

func (t *mailer) openAppDatabase() (*sqldb.DB, error) {
	return sqlite.NewDB(t.log, t.sqliteDir, sqlite.DBDefault, false)
}

func (t *mailer) sendUnsentEmails() error {
	unsentEmails, err := t.selectAllUnsentEmails()
	if err != nil {
		return err
	}
	for _, email := range unsentEmails {
		err = t.sendEmail(email)
		if err != nil {
			t.log.Error(t.prefix+"Error while sending email", "error", err)
		}
	}
	return nil
}

func (t *mailer) selectAllUnsentEmails() ([]*model.Email, error) {
	// open local sqlite db
	db, err := t.openAppDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// select all unsent emails
	var result = []*model.Email{}
	if err := db.Select(&result, "select * from email where sent IS NULL AND (retries IS NULL OR retries <= 3) order by created desc"); err != nil {
		return nil, err
	}
	return result, nil
}

func (t *mailer) sendEmail(em *model.Email) error {
	var sender smtp.Smtp
	// var gmail, ok = t.smtps[smtp.GMAIL]
	// if !ok {
	// 	return fmt.Errorf("No %s smtp provider found!", smtp.GMAIL)
	// }
	mailerSend, ok := t.smtps[smtp.MAILSENDER]
	if !ok {
		return fmt.Errorf("No %s smtp provider found!", smtp.MAILSENDER)
	}
	switch em.Type {
	default:
		sender = mailerSend
	}
	if e := sender.SendEmail("", em.To, em.Subject, "text/html", em.BodyHtml, nil); e != nil {
		t.log.Error("Error while sending out email", "sender", sender.Provider(), "to", em.To, "subject", em.Subject, "error", e)

		// on error mark email sending failed
		db, err := t.openAppDatabase()
		if err != nil {
			return err
		}
		defer db.Close()

		// increase the retry count & mark the last error message
		if em.Retries == nil {
			var r int = 0
			em.Retries = &r
		}
		*em.Retries++
		var errVal = e.Error()
		em.LastError = &errVal

		// update the email
		_, err = db.Update(em, []string{"Retries", "LastError"})
		if err != nil {
			return err
		}
		return nil
	}

	// on success mark email sent
	db, err := t.openAppDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// set the Sent date
	var now = time.Now()
	em.Sent = &now

	// update the email
	_, err = db.Update(em, []string{"Sent"})
	if err != nil {
		return err
	}
	return nil
}

func (t *mailer) StoreEmail(e *model.Email) error {
	// open local sqlite db
	db, err := t.openAppDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// store email in db
	r, err := db.Insert(e, []string{"id"})
	if err != nil {
		return err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = uint(id)
	return nil
}

func (t *mailer) GetEmailTemplate(code TemplateCode, lang string) (*model.EmailTemplate, error) {
	// open local sqlite db
	db, err := t.openAppDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// select email template
	var result = []*model.EmailTemplate{}
	if err := db.Select(&result, "select t.code, tr.subject, tr.body from emailTemplates t join emailTemplates_translations tr on tr.emailTemplates_code = t.code and tr.languages_code = ? and t.code = ?", lang, code); err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result[0], nil
	}
	return nil, nil
}

type GenerateEmailDataInput interface {
	Title()
}

func GenerateEmail(tce i18n.Translations, templateName string, tmpl model.EmailTemplate, data interface{}) (model.EmailTemplate, error) {
	var fm = template.FuncMap{
		"i18n": func(key string, replacements ...string) string {
			if tce == nil {
				return ""
			}
			return i18n.Translate(tce, key, replacements...)
		},
	}
	subject, err := html.Generate(tmpl.Subject, template.FuncMap{}, data)
	if err != nil {
		return model.EmailTemplate{}, fmt.Errorf("generate template subject error: %w, subject: %v", err, tmpl.Subject)
	}
	body, err := html.Generate(tmpl.Body, template.FuncMap{}, data)
	if err != nil {
		return model.EmailTemplate{}, fmt.Errorf("generate template body error: %w, body: %v", err, tmpl.Body)
	}

	// parse json to a content struct
	var content = BlockEditorContent{}
	err = json.Unmarshal([]byte(body), &content)
	if err != nil {
		return model.EmailTemplate{}, fmt.Errorf("unmarshal body content error: %w, body: %v", err, body)
	}

	// iterate over all blocks & generate html out of it
	var builder strings.Builder
	for _, b := range content.Blocks {
		switch b.Type {
		case "header":
			var level = strconv.Itoa(b.Data.Level)
			builder.WriteString("<h" + level + ">")
			builder.WriteString(b.Data.Text)
			builder.WriteString("</h" + level + ">")
		case "paragraph":
			builder.WriteString("<p>")
			builder.WriteString(b.Data.Text)
			builder.WriteString("</p>")
		case "nestedlist":
			var tag = "ol"
			var style = b.Data.Style
			if style == "unordered" {
				tag = "ul"
			}
			builder.WriteString("<" + tag + ">")
			for _, item := range b.Data.Items {
				builder.WriteString("<li>")
				builder.WriteString(item.Content)
				builder.WriteString("</li>")
			}
			builder.WriteString("</" + tag + ">")
		}
	}

	// wrap a body with general email template
	if templateName == "" {
		templateName = "email-general.html"
	}
	wrappedBody, err := html.GenerateFS(html.Templates, "templates", templateName, fm, html.GeneralEmail{
		Body: builder.String(),
	})
	if err != nil {
		return model.EmailTemplate{}, fmt.Errorf("generate html error: %w. template name: %v, body: %v", err, templateName, builder.String())
	}
	return model.EmailTemplate{
		Code:    tmpl.Code,
		Subject: subject,
		Body:    wrappedBody,
	}, nil
}

type BlockEditorContent struct {
	Time   uint64             `json:"time"`
	Blocks []BlockEditorBlock `json:"blocks"`
}

type BlockEditorBlock struct {
	ID   string               `json:"id"`
	Type string               `json:"type"`
	Data BlockEditorDataUnion `json:"data"`
}

type BlockEditorDataUnion struct {
	// nestedList properties
	Style string                `json:"style"`
	Items []BlockEditorDataItem `json:"items"`
	// header && paragraph
	Text  string `json:"text"`
	Level int    `json:"level"`
	// image
	File BlockEditorImageFile `json:"file"`
}

type BlockEditorDataItem struct {
	// nestedList properties
	Content string                `json:"content"`
	Items   []BlockEditorDataItem `json:"items"`
	// checklist
	Text    string `json:"text"`
	Checked bool   `json:"checked"`
}

type BlockEditorImageFile struct {
	Width          int    `json:"width"`
	Height         int    `json:"height"`
	Size           int    `json:"size"`
	Name           string `json:"name"`
	Title          string `json:"title"`
	Ext            string `json:"extension"`
	FileId         string `json:"fileId"`
	FileURL        string `json:"fileURL"`
	URL            string `json:"url"`
	Caption        string `json:"caption"`
	WithBorder     bool   `json:"withBorder"`
	Stretched      bool   `json:"stretched"`
	WithBackground bool   `json:"withBackground"`
}
