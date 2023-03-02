package mq

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

func NewMqttClient(log logf.Logger, id string, uri *url.URL, handlers func() []MessageHandler) MqttClient {
	// Create client options
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s", uri.Host))
	opts.SetUsername(uri.User.Username())
	pw, isSet := uri.User.Password()
	if isSet {
		opts.SetPassword(pw)
	}
	opts.SetClientID(id)
	opts.SetPingTimeout(10 * time.Second)
	opts.SetKeepAlive(10 * time.Second)

	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(10 * time.Second)
	opts.SetConnectionLostHandler(func(c mqtt.Client, err error) {
		log.Info(fmt.Sprintf("[mqtt] connection lost error: %s\n" + err.Error()))
	})
	opts.SetReconnectingHandler(func(c mqtt.Client, options *mqtt.ClientOptions) {
		log.Info("[mqtt] reconnecting ......")
	})
	// for example if mqtt server is restarted then we auto-reconnect and re-subscribe listeners
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		if handlers == nil {
			return
		}
		var handlers = handlers()
		log.Info(fmt.Sprintf("[mqtt] (re)connected. (re)subscribing message handlers (%v) ...", len(handlers)))
		for _, h := range handlers {
			if token := c.Subscribe(h.Topic(), 0, func(c mqtt.Client, m mqtt.Message) {
				err := h.Callback(m)
				if err != nil {
					log.Warn(fmt.Sprintf("[mqtt] handler message listener error (topic: %s): %v", h.Topic(), err))
				}
			}); token.Wait() && token.Error() != nil {
				log.Warn(fmt.Sprintf("[mqtt] handler subscribe error (topic: %s): %v", h.Topic(), token.Error()))
			} else {
				log.Info(fmt.Sprintf("[mqtt] handler (re)subscribed (topic: %s)", h.Topic()))
			}
		}
		log.Info(fmt.Sprintf("[mqtt] (re)subscribed message handlers (%v) ...", len(handlers)))
	})

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

// func (l *listener) Start(ctx context.Context) {
// 	// TEST - Subscribe to sensors sending to mqtt
// 	// "zigbee2mqtt/Katel-toru-vesi1"
// 	// Start mqtt source subscriptions listeners
// 	_, err = l.mq.Subscribe(ctx, 1*time.Minute, "zigbee2mqtt/Ikea-switch1-stainless-4button", func(client mqtt.Client, msg mqtt.Message) {
// 		l.lo.Info("[MQTT] message", "topic", msg.Topic(), "payload", string(msg.Payload()))
// 	})
// 	if err != nil {
// 		return err
// 	}
// }
