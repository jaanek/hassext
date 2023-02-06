package hass

import (
	"context"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jaanek/hassext/emodul"
	"github.com/jaanek/hassext/mq"
	"github.com/jaanek/hassext/rest"
	"github.com/knadh/koanf"
	"github.com/zerodha/logf"
)

type HassExt struct {
	opts   *Options
	Lo     logf.Logger
	Mq     mq.MqttClient
	Emodul emodul.EModul
	Rest   *rest.Rest
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
	em := emodul.NewEmodulClient(lo, mq, &emodul.HttpClientParams{
		SkipRetryAuthorization: false,
		ApiUrl:                 ko.String("emodul.apiUrl"),
		FrontendUrl:            ko.String("emodul.frontendUrl"),
		Username:               ko.String("emodul.username"),
		Password:               ko.String("emodul.password"),
		ModuleHash:             ko.String("emodul.moduleid"),
		ModuleIndex:            0,
		Cookies:                map[string]string{},
	})
	r := rest.NewRest(lo, em, ko.String("rest.host"), ko.Int("rest.port"), ko.String("rest.jwtSecret"))

	return &HassExt{
		opts:   opts,
		Lo:     lo,
		Mq:     mq,
		Emodul: em,
		Rest:   r,
	}, nil
}

func (h *HassExt) Run(ctx context.Context) error {
	// ct, c := context.WithCancel(context.Background())
	_, err := h.Mq.Connect(context.Background(), 1*time.Minute)
	if err != nil {
		return err
	}

	// TEST - Subscribe to sensors sending to mqtt
	_, err = h.Mq.Subscribe(ctx, 1*time.Minute, "zigbee2mqtt/Katel-toru-vesi1", func(client mqtt.Client, msg mqtt.Message) {
		h.Lo.Info("[MQTT] message", "topic", msg.Topic(), "payload", string(msg.Payload()))
	})
	if err != nil {
		return err
	}

	// Start emodul
	if err = h.Emodul.Init(); err != nil {
		h.Lo.Error("eModul init", "failed", err)
		return err
	}
	go func() {
		h.Emodul.Start(ctx)
	}()

	// Start rest api
	go func() {
		h.Rest.Start(ctx)
	}()

	return nil
}

func (h *HassExt) Shutdown() {
	h.Mq.Disconnect()
	h.Rest.Shutdown(context.Background())
}
