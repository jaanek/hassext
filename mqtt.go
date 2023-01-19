package hass

import (
	"context"
	"fmt"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zerodha/logf"
)

type MqttClient interface {
	Connect(context.Context, time.Duration) (mqtt.Token, error)
	Subscribe(context.Context, time.Duration, string, mqtt.MessageHandler) (mqtt.Token, error)
	Publish(context.Context, time.Duration, string, any) error
	Disconnect()
}

type client struct {
	log logf.Logger
	c   mqtt.Client
}

func NewMqttClient(log logf.Logger, id string, uri *url.URL) MqttClient {
	// Create client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	pw, isSet := uri.User.Password()
	if isSet {
		opts.SetPassword(pw)
	}
	opts.SetClientID(id)

	// Create client
	return &client{
		log: log,
		c:   mqtt.NewClient(opts),
	}
}

func (m *client) Connect(ctx context.Context, timeout time.Duration) (mqtt.Token, error) {
	token := m.c.Connect()

	// Wait connection to complete
	return token, waitOnToken(m.log, ctx, timeout, token)
}

func (m *client) Subscribe(ctx context.Context, timeout time.Duration, topic string, callback mqtt.MessageHandler) (mqtt.Token, error) {
	token := m.c.Subscribe(topic, 0, callback)

	// Wait to complete
	return token, waitOnToken(m.log, ctx, timeout, token)
}

func (m *client) Publish(ctx context.Context, timeout time.Duration, topic string, payload any) error {
	token := m.c.Publish(topic, 0, false, payload)

	// Wait a send to complete
	return waitOnToken(m.log, ctx, timeout, token)
}

func (m *client) Disconnect() {
	m.c.Disconnect(100)
}

func waitOnToken(log logf.Logger, ctx context.Context, timeout time.Duration, token mqtt.Token) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		if !timer.Stop() {
			<-timer.C
		}
		log.Warn("mqtt timeout")
		return nil
	case <-token.Done():
		if token.Error() != nil {
			return token.Error()
		}
		log.Debug("mqtt success")
		return nil
	case <-ctx.Done():
		timer.Stop()
		log.Warn("mqtt cancelled")
		return nil
	}
}
