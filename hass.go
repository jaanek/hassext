package hass

import (
	"context"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jaanek/hassext/emodul"
	"github.com/jaanek/hassext/hub"
	"github.com/jaanek/hassext/mq"
	"github.com/jaanek/hassext/rest"
	"github.com/knadh/koanf"
	"github.com/zerodha/logf"
)

type HassExt struct {
	opts   *Options
	Lo     logf.Logger
	Hub    *hub.Hub
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
	hub := hub.New(lo)
	mq := mq.NewMqttClient(lo, "hassext", uri, MessageHandlers(lo, hub))
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
		Hub:    hub,
		Mq:     mq,
		Emodul: em,
		Rest:   r,
	}, nil
}

func (h *HassExt) Run(ctx context.Context) error {
	// start the message hub
	go func() {
		h.Hub.Run(ctx)
	}()
	// connect to the mq so that messages start flowing to the hub
	_, err := h.Mq.Connect(ctx, 30*time.Second)
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

func MessageHandlers(lo logf.Logger, h *hub.Hub) func() []mq.MessageHandler {
	return func() []mq.MessageHandler {
		return []mq.MessageHandler{
			mq.NewHandler(lo, "zigbee2mqtt/Ikea-switch1-stainless-4button", func(m mqtt.Message) error {
				var payload = m.Payload()
				h.Broadcast <- hub.Message{
					Topic: "sound-switch-leiliruum",
					Data:  payload,
				}
				return nil
			}),
			mq.NewHandler(lo, "zigbee2mqtt/Ikea-switch2-white-4button", func(m mqtt.Message) error {
				var payload = m.Payload()
				h.Broadcast <- hub.Message{
					Topic: "sound-switch-sauna-eesruum",
					Data:  payload,
				}
				return nil
			}),
		}
	}
}
