package mq

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/zerodha/logf"
)

type MessageCallbackFunc func(mqtt.Message) error

type MessageHandler interface {
	Topic() string
	Callback(mqtt.Message) error
}

type handler struct {
	lo       logf.Logger
	topic    string
	callback MessageCallbackFunc
}

func NewHandler(lo logf.Logger, topic string, callback MessageCallbackFunc) MessageHandler {
	return &handler{
		lo:       lo,
		topic:    topic,
		callback: callback,
	}
}

func (l *handler) Topic() string {
	return l.topic
}

func (l *handler) Callback(m mqtt.Message) error {
	return l.callback(m)
}
