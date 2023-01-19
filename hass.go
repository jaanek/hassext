package hass

import (
	"context"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/knadh/koanf"
	"github.com/zerodha/logf"
)

type HassExt struct {
	opts   *Options
	lo     logf.Logger
	mq     MqttClient
	emodul EModul
}

// init home assistant integration
func Init(ko *koanf.Koanf, lo logf.Logger) (*HassExt, error) {
	// Set options
	opts := DefaultOptions()

	// mqtt client url
	uri, err := url.Parse(ko.String("mqtt.url"))
	if err != nil {
		return nil, err
	}
	mq := NewMqttClient(lo, "hassext", uri)

	return &HassExt{
		opts:   opts,
		lo:     lo,
		mq:     mq,
		emodul: NewEmodulClient(lo, mq, ko.String("emodul.url"), ko.String("emodul.token")),
	}, nil
}

func (h *HassExt) Run(ctx context.Context) error {
	// ct, c := context.WithCancel(context.Background())
	_, err := h.mq.Connect(context.Background(), 1*time.Minute)
	if err != nil {
		return err
	}

	// Subscribe to sensors sending to mqtt
	_, err = h.mq.Subscribe(ctx, 1*time.Minute, "zigbee2mqtt/Katel-toru-vesi1", func(client mqtt.Client, msg mqtt.Message) {
		h.lo.Info("[MQTT] message", "topic", msg.Topic(), "payload", string(msg.Payload()))
	})
	if err != nil {
		return err
	}

	// Start emodul
	go func() {
		h.emodul.Start(ctx)
	}()

	return nil
}

func (h *HassExt) Shutdown() {
	h.mq.Disconnect()
}
