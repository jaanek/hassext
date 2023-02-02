package hass

import (
	"context"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jaanek/hassext/emodul"
	"github.com/jaanek/hassext/mq"
	"github.com/knadh/koanf"
	"github.com/zerodha/logf"
)

type HassExt struct {
	opts   *Options
	lo     logf.Logger
	mq     mq.MqttClient
	emodul emodul.EModul
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
	mq := mq.NewMqttClient(lo, "hassext", uri)

	return &HassExt{
		opts: opts,
		lo:   lo,
		mq:   mq,
		emodul: emodul.NewEmodulClient(lo, mq, &emodul.HttpClientParams{
			SkipRetryAuthorization: false,
			Url:                    ko.String("emodul.url"),
			Username:               ko.String("emodul.username"),
			Password:               ko.String("emodul.password"),
			ModuleId:               ko.String("emodul.moduleid"),
		}),
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
	if err = h.emodul.Init(); err != nil {
		h.lo.Error("eModul init", "failed", err)
		return err
	}
	go func() {
		h.emodul.Start(ctx)
	}()

	return nil
}

func (h *HassExt) Shutdown() {
	h.mq.Disconnect()
}
