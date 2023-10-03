package sound

import (
	"encoding/json"

	"github.com/jaanek/hassext/hub"
	"github.com/jaanek/hassext/snapcast"
	"github.com/zerodha/logf"
)

const (
	TopicLeiliruumSoundButtons    = "sound-buttons-leiliruum"
	TopicSaunaEesruumSoundButtons = "sound-buttons-sauna-eesruum"
	TopicElutubaTvSoundButtons    = "sound-buttons-elutuba-tv"
	ClientIdLeiliruum             = "zone-leiliruum"
	ClientIdSaunaEesruum          = "zone-sauna-eesruum"
	ClientIdElutubaTv             = "zone-elutuba"
)

type Sound interface {
	Init()
	Shutdown()
}

type SoundListener interface {
	hub.Client
	Topic() string
}

type IkeaButtons struct {
	lo               logf.Logger
	snapcast         snapcast.Snapcast
	enabled          bool
	listenTopic      string
	snapcastClientId string
}

func (s *IkeaButtons) Topic() string {
	return s.listenTopic
}

func (s *IkeaButtons) Receive(data []byte) error {
	if !s.enabled {
		s.lo.Info("Sound switch. On data receive. Switch disabled", "topic", s.listenTopic, "name", s.snapcastClientId)
		return nil
	}
	msg := &IkeaSwitchMessage{}
	err := json.Unmarshal(data, msg)
	if err != nil {
		return err
	}
	switch msg.Action {
	case "brightness_move_up":
		// "top" long press. Mute or unmute
		s.snapcast.SendRequest(snapcast.MuteOnOffReq{
			ClientId: s.snapcastClientId,
		})
	case "brightness_move_down":
		// "down" long press. Set default stream channel
		var streamId = ""
		switch s.snapcastClientId {
		case ClientIdLeiliruum:
			streamId = "SpaMusic"
		case ClientIdSaunaEesruum:
			streamId = "BirdRelaxMusic"
		case ClientIdElutubaTv:
			streamId = "SkyPlus"
		default:
			streamId = "SkyPlus"
		}
		s.snapcast.SendRequest(snapcast.SetDefaultChannelReq{
			ClientId: s.snapcastClientId,
			StreamId: streamId,
		})
	case "arrow_right_click":
		s.snapcast.SendRequest(snapcast.IncVolumeReq{
			ClientId: s.snapcastClientId,
			IncStep:  2,
		})
	case "arrow_left_click":
		s.snapcast.SendRequest(snapcast.IncVolumeReq{
			ClientId: s.snapcastClientId,
			IncStep:  -2,
		})
	case "on":
		// top single press
		s.snapcast.SendRequest(snapcast.ChangeStreamReq{
			ClientId: s.snapcastClientId,
			Up:       true,
		})
	case "off":
		// bottom single press
		s.snapcast.SendRequest(snapcast.ChangeStreamReq{
			ClientId: s.snapcastClientId,
			Up:       false,
		})
	}
	return nil
}

type sound struct {
	hub       *hub.Hub
	listeners []SoundListener
}

func New(lo logf.Logger, h *hub.Hub, snapcast snapcast.Snapcast) Sound {
	return &sound{
		hub: h,
		listeners: []SoundListener{
			&IkeaButtons{
				lo:               lo,
				snapcast:         snapcast,
				enabled:          true,
				listenTopic:      TopicLeiliruumSoundButtons,
				snapcastClientId: ClientIdLeiliruum,
			},
			&IkeaButtons{
				lo:               lo,
				snapcast:         snapcast,
				enabled:          true,
				listenTopic:      TopicSaunaEesruumSoundButtons,
				snapcastClientId: ClientIdSaunaEesruum,
			},
			&IkeaButtons{
				lo:               lo,
				snapcast:         snapcast,
				enabled:          true,
				listenTopic:      TopicElutubaTvSoundButtons,
				snapcastClientId: ClientIdElutubaTv,
			},
		},
	}
}

func (s *sound) Init() {
	// register into the hub
	for _, listener := range s.listeners {
		s.hub.Register <- hub.Registration{
			Topic:  listener.Topic(),
			Client: listener,
		}
	}
}

func (s *sound) Shutdown() {
	// unregister from hub
	for _, listener := range s.listeners {
		s.hub.Unregister <- hub.Registration{
			Topic:  listener.Topic(),
			Client: listener,
		}
	}
}
