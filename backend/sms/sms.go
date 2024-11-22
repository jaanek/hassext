package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"sync"
	"time"

	"github.com/jaanek/hassext/html"
	"github.com/jaanek/hassext/httpclient"
	"github.com/jaanek/hassext/mailer"
	"github.com/jaanek/hassext/model"
	"github.com/jaanek/hassext/sqldb"
	"github.com/jaanek/hassext/sqlite"

	"github.com/zerodha/logf"
)

type Sender interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	StoreSMS(e *model.SMS) error
	TriggerUnsentSMS() error
}

type CheckSMS struct{}

type sender struct {
	prefix       string
	log          logf.Logger
	url          string
	token        string
	http         httpclient.HttpClient
	params       *HttpClientParams
	done         chan struct{}
	doneOnce     sync.Once
	triggerCheck chan CheckSMS
	sqliteDir    string
	mailer       mailer.Mailer
}

type HttpClientParams struct {
}

func New(log logf.Logger, params *HttpClientParams, url, token string, dataDir string, mailer mailer.Mailer) Sender {
	return &sender{
		prefix: "[sms] ",
		log:    log,
		http: httpclient.New(
			nil,
			func(req *httpclient.Request, resp *http.Response, err error) (bool, error) {
				return false, nil
			},
			func(attemptNum int, resp *http.Response) (waitDelay time.Duration) {
				return 0
			},
			false,
		),
		params:       params,
		url:          url,
		token:        token,
		done:         make(chan struct{}),
		triggerCheck: make(chan CheckSMS), // unbuffered because one at a time sms sending & sequential poll from sms table
		sqliteDir:    dataDir,
		mailer:       mailer,
	}
}

func (s *sender) sendSMS(to string, body string) error {
	var result = struct {
		Data struct {
			UUID string `json:"uuid"`
		} `json:"data"`
	}{}
	var input = struct {
		To   string `json:"to"`
		Body string `json:"body"`
	}{
		To:   to,
		Body: body,
	}
	var jsonBytes, err = json.Marshal(input)
	if err != nil {
		return err
	}
	_, err = httpclient.Post(s.http, s.url, jsonBytes, func(req *httpclient.Request) {
		fmt.Printf("[REQUEST] POST: %v, data: %v\n", s.url, string(jsonBytes))
		req.Header.Set("Authorization", "Bearer "+s.token)
	}, func(resp *http.Response) ([]byte, error) {
		body, e := httpclient.ReadJsonResult(resp, &result)
		fmt.Println(fmt.Sprintf("[RESPONSE] Status code: %d, status: %s, result: %v", resp.StatusCode, resp.Status, string(body)))
		if e != nil {
			return nil, e
		}
		return body, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (t *sender) TriggerUnsentSMS() error {
	t.triggerCheck <- CheckSMS{}
	return nil
}

func (t *sender) Start(ctx context.Context) (err error) {
	t.log.Info(t.prefix + "Starting hassext SMS Sender")

	// start polling loop or react to triggering event
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-t.triggerCheck:
				err := t.triggerUnfinishedSMS()
				if err != nil {
					t.log.Error(t.prefix+"error while triggering to send sms", "error", err)
				}
			case <-ticker.C:
				err := t.triggerUnfinishedSMS()
				if err != nil {
					t.log.Error(t.prefix+"error while triggering to send sms", "error", err)
				}
			case <-t.done:
				t.log.Info(t.prefix + "Exiting hassext sms sender ...")
				return
			}
		}
	}()
	return nil
}

func (t *sender) Stop(ctx context.Context) error {
	t.log.Info(t.prefix + "Stopping hassext SMS Sender")
	t.doneOnce.Do(func() {
		close(t.done)
	})
	return nil
}

func (t *sender) openAppDatabase() (*sqldb.DB, error) {
	return sqlite.NewDB(t.log, t.sqliteDir, sqlite.DBDefault, false)
}

func (t *sender) triggerUnfinishedSMS() error {
	unfinishedSMSs, err := t.selectAllUnfinishedSMS()
	if err != nil {
		return err
	}
	for _, sms := range unfinishedSMSs {
		err = t.triggerSMS(sms)
		if err != nil {
			t.log.Error(t.prefix+"Error while triggering to send sms", "error", err)
		}
	}
	return nil
}

func (t *sender) selectAllUnfinishedSMS() ([]*model.SMS, error) {
	// open local sqlite db
	db, err := t.openAppDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// select all unfinished calls
	var result = []*model.SMS{}
	if err := db.Select(&result, "select * from sms where finished IS NULL AND (retries IS NULL OR retries <= 10) order by created desc"); err != nil {
		return nil, err
	}
	return result, nil
}

func (t *sender) triggerSMS(em *model.SMS) error {
	// open local sqlite db
	var db, err = t.openAppDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// send sms
	err = t.sendSMS(em.To, em.Message)
	if err != nil {
		t.log.Error("Error while sending sms", "to", em.To, "message", em.Message, "error", err)
		var merr = t.SendSmsFailedEmail("jaanekoja@gmail.com")
		if merr != nil {
			t.log.Error("Failed to send sms sending failed email!", merr)
		}

		// increase the retry count & mark the last error message
		if em.Retries == nil {
			var r int = 0
			em.Retries = &r
		}
		*em.Retries++
		var errStr = err.Error()
		em.Result = &errStr

		// update the sms
		_, err = db.Update(em, []string{"Retries", "Result"})
		if err != nil {
			return err
		}
		return nil
	}

	// set the Finished date
	var now = time.Now()
	em.Finished = &now

	// update the call
	_, err = db.Update(em, []string{"Finished"})
	if err != nil {
		return err
	}
	return nil
}

func (t *sender) StoreSMS(e *model.SMS) error {
	// open local sqlite db
	db, err := t.openAppDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	// store sms in db
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

func (t *sender) SendSmsFailedEmail(toEmail string) error {
	var input = struct{}{}
	var fm = template.FuncMap{}
	body, err := html.GenerateFS(html.Templates, "templates", "sms-sending-failed.html", fm, input)
	if err != nil {
		return fmt.Errorf("failed to generate sms sending failed email body: %v", err)
	}
	var email = model.NewEmail(model.SMS_SENDING_FAILED, "", toEmail, "[Alert] hassext. SMS sending failed!", body)
	t.log.Info("Generated sms sending failed email", "email", email)
	if err = t.mailer.StoreEmail(email); err != nil {
		fmt.Printf(t.prefix+"failed to store sms sending failed email: %v", err)
	}
	return nil
}
