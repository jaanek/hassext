package hass

import (
	"context"
	"net/url"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jaanek/hassext/brain"
	"github.com/jaanek/hassext/emodul"
	"github.com/jaanek/hassext/homeassistant"
	"github.com/jaanek/hassext/hub"
	"github.com/jaanek/hassext/mq"
	"github.com/jaanek/hassext/rest"
	"github.com/jaanek/hassext/snapcast"
	"github.com/jaanek/hassext/sound"
	"github.com/knadh/koanf"
	"github.com/zerodha/logf"
)

type HassExt struct {
	opts     *Options
	Lo       logf.Logger
	Hub      *hub.Hub
	Mq       mq.MqttClient
	Emodul   emodul.EModul
	Snapcast snapcast.Snapcast
	Rest     *rest.Rest
	Sound    sound.Sound
	HA       homeassistant.HomeAssistant
	Brain    brain.Brain
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
	sc := snapcast.New(lo, &snapcast.HttpClientParams{
		ApiUrl: ko.String("snapcast.apiUrl"),
	})
	r := rest.NewRest(lo, em, sc, ko.String("rest.host"), ko.Int("rest.port"), ko.String("rest.jwtSecret"))
	sound := sound.New(lo, hub, sc)
	ha := homeassistant.NewHomeAssistantClient(lo, &homeassistant.HttpClientParams{
		ApiUrl: ko.String("homeassistant.apiUrl"),
		Token:  ko.String("homeassistant.token"),
	})
	brain := brain.NewBrain(lo, ha)

	return &HassExt{
		opts:     opts,
		Lo:       lo,
		Hub:      hub,
		Mq:       mq,
		Emodul:   em,
		Snapcast: sc,
		Rest:     r,
		Sound:    sound,
		HA:       ha,
		Brain:    brain,
	}, nil
}

func (h *HassExt) Run(ctx context.Context) error {
	// start the message hub
	go func() {
		h.Hub.Run(ctx)
	}()

	// start sound listener
	go func() {
		h.Sound.Init()
	}()

	// Start Snapcast
	go func() {
		h.Snapcast.Run(ctx)
	}()

	go func() {
		h.Brain.Run(ctx)
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

	// Start rest server api
	go func() {
		h.Rest.Start(ctx)
	}()

	return nil
}

func (h *HassExt) Shutdown() {
	h.Lo.Info("Hass shutting down ...")
	// h.Sound.Shutdown()
	h.Mq.Disconnect()
	h.Rest.Shutdown(context.Background())
	h.Lo.Info("Hass shutdown success")
}

func MessageHandlers(lo logf.Logger, h *hub.Hub) func() []mq.MessageHandler {
	return func() []mq.MessageHandler {
		return []mq.MessageHandler{
			// mq.NewHandler(lo, "zigbee2mqtt/Leiliruum-lights-power", func(m mqtt.Message) error {
			// 	var payload = m.Payload()
			// 	h.Broadcast <- hub.Message{
			// 		Topic: sound.TopicLeiliruumLightsPower,
			// 		Data:  payload,
			// 	}
			// 	return nil
			// }),
			mq.NewHandler(lo, "zigbee2mqtt/Ikea-switch1-stainless-4button", func(m mqtt.Message) error {
				var payload = m.Payload()
				h.Broadcast <- hub.Message{
					Topic: sound.TopicLeiliruumSoundButtons,
					Data:  payload,
				}
				return nil
			}),
			mq.NewHandler(lo, "zigbee2mqtt/Ikea-switch2-white-4button", func(m mqtt.Message) error {
				var payload = m.Payload()
				h.Broadcast <- hub.Message{
					Topic: sound.TopicSaunaEesruumSoundButtons,
					Data:  payload,
				}
				return nil
			}),
			mq.NewHandler(lo, "zigbee2mqtt/Ikea-switch3-white-4button", func(m mqtt.Message) error {
				var payload = m.Payload()
				h.Broadcast <- hub.Message{
					Topic: sound.TopicElutubaTvSoundButtons,
					Data:  payload,
				}
				return nil
			}),
		}
	}
}
