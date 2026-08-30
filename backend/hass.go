package hass

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/jaanek/hassext/brain"
	"github.com/jaanek/hassext/chromecast"
	"github.com/jaanek/hassext/emodul"
	"github.com/jaanek/hassext/floorheating"
	"github.com/jaanek/hassext/homeassistant"
	"github.com/jaanek/hassext/hub"
	"github.com/jaanek/hassext/mailer"
	"github.com/jaanek/hassext/mq"
	"github.com/jaanek/hassext/rest"
	"github.com/jaanek/hassext/sms"
	"github.com/jaanek/hassext/smtp"
	"github.com/jaanek/hassext/snapcast"
	"github.com/jaanek/hassext/sound"
	"github.com/jaanek/hassext/spotify"
	"github.com/jaanek/hassext/sqlite"
	"github.com/jaanek/hassext/uponor"
	"github.com/knadh/koanf"
	"github.com/zerodha/logf"
)

type HassExt struct {
	opts         *Options
	Lo           logf.Logger
	Hub          *hub.Hub
	Mq           mq.MqttClient
	Emodul       emodul.EModul
	Snapcast     snapcast.Snapcast
	Rest         *rest.Rest
	Sound        sound.Sound
	HA           homeassistant.HomeAssistant
	Brain        brain.Brain
	Chromecasts  chromecast.Chromecasts
	Spotify      spotify.Spotify
	Mailer       mailer.Mailer
	SmsSender    sms.Sender
	FloorHeating floorheating.FloorHeating
}

// init home assistant integration
func Init(ko *koanf.Koanf, lo logf.Logger) (*HassExt, error) {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}
	env = strings.ToLower(env)
	env = strings.TrimSpace(env)
	var isProd = env == "production"
	if isProd {
		lo.Info(fmt.Sprintf("*************** Running in %s ********************", env))
	}

	// Set options
	opts := DefaultOptions()

	// create/open the sqlite databases
	var dataDir = ko.String("sqldb.dataDir")
	appDb, err := sqlite.NewDB(lo, dataDir, "default", true)
	if err != nil {
		return nil, fmt.Errorf("Error while opening sqlite database: %v. Error: %w", "default", err)
	}
	appDb.Close() // we do not need the instance here

	// mailer
	mailersendSmtp := smtp.New(lo, smtp.MAILSENDER, ko.String("smtp.host"), ko.Int("smtp.port"), ko.String("smtp.username"), ko.String("smtp.password"), ko.String("smtp.fromEmail"), ko.String("smtp.fromName"))
	mailer := mailer.New(lo, dataDir, mailersendSmtp)
	smsSender := sms.New(lo, &sms.HttpClientParams{}, ko.String("sms.url"), ko.String("sms.token"), dataDir, mailer)

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
	cert, err := loadCert()
	if err != nil {
		return nil, fmt.Errorf("Error loading client certs: %w", err)
	}
	chromecasts := chromecast.NewDevices(lo, "", ko.Strings("chromecast.devices"), &cert)
	// Spotify Web API client: drives the Spotify Connect receiver (librespot) in
	// the living room from the IKEA sound remote. Optional.
	var sp spotify.Spotify
	if ko.String("spotify.clientId") != "" {
		sp = spotify.New(lo, &spotify.Params{
			ClientId:     ko.String("spotify.clientId"),
			ClientSecret: ko.String("spotify.clientSecret"),
			RefreshToken: ko.String("spotify.refreshToken"),
			DataDir:      dataDir,
		})
	} else {
		lo.Warn("[spotify] not configured (spotify.clientId is empty): living room remote keeps the snapcast only behaviour")
	}
	spotifyOpts := sound.SpotifyOptions{
		DeviceName:      ko.String("spotify.deviceName"),
		DataDir:         dataDir,
		MorningPlaylist: ko.String("spotify.morningPlaylist"),
		MorningFrom:     ko.String("spotify.morningFrom"),
		MorningTo:       ko.String("spotify.morningTo"),
	}
	if spotifyOpts.MorningFrom == "" {
		spotifyOpts.MorningFrom = "06:00"
	}
	if spotifyOpts.MorningTo == "" {
		spotifyOpts.MorningTo = "09:30"
	}
	sound := sound.New(lo, hub, sc, chromecasts, sp, spotifyOpts)
	ha := homeassistant.NewHomeAssistantClient(lo, &homeassistant.HttpClientParams{
		ApiUrl: ko.String("homeassistant.apiUrl"),
		Token:  ko.String("homeassistant.token"),
	})
	uponorClient := uponor.NewUponorClient(lo, &uponor.HttpClientParams{
		Host: ko.String("uponor.host"),
	})
	flootHeating := floorheating.New(lo, dataDir, mq, ha)
	brain := brain.NewBrain(lo, ha, mq, uponorClient, dataDir, flootHeating)

	return &HassExt{
		opts:         opts,
		Lo:           lo,
		Hub:          hub,
		Mq:           mq,
		Emodul:       em,
		Snapcast:     sc,
		Rest:         r,
		Sound:        sound,
		HA:           ha,
		Brain:        brain,
		Chromecasts:  chromecasts,
		Spotify:      sp,
		Mailer:       mailer,
		SmsSender:    smsSender,
		FloorHeating: flootHeating,
	}, nil
}

func (h *HassExt) Run(ctx context.Context) error {
	// start the message hub
	go func() {
		h.Hub.Run(ctx)
	}()

	err := h.Mailer.Start(ctx)
	if err != nil {
		return err
	}
	h.SmsSender.Start(ctx)

	// start sound listener
	go func() {
		h.Sound.Init()
	}()
	go func() {
		h.Sound.Run(ctx)
	}()

	// Start Snapcast
	go func() {
		h.Snapcast.Run(ctx)
	}()

	go func() {
		h.Brain.Run(ctx)
	}()

	// connect to the mq so that messages start flowing to the hub
	_, err = h.Mq.Connect(ctx, 30*time.Second)
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

	// Connect & keep connections to defined chromecasts
	go func() {
		h.Chromecasts.Start(ctx)
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
	h.Chromecasts.Stop(context.Background())
	h.Rest.Shutdown(context.Background())
	h.Mailer.Stop(context.Background())
	h.SmsSender.Stop(context.Background())
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
			// mq.NewHandler(lo, "zigbee2mqtt/Ikea-switch3-white-4button", func(m mqtt.Message) error {
			// 	var payload = m.Payload()
			// 	h.Broadcast <- hub.Message{
			// 		Topic: sound.TopicElutubaTvSoundButtons,
			// 		Data:  payload,
			// 	}
			// 	return nil
			// }),
			mq.NewHandler(lo, "zigbee2mqtt/ikea-sound-remote-elutuba", func(m mqtt.Message) error {
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
