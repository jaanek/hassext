package sound

import (
	"encoding/json"

	"github.com/jaanek/hassext/hub"
	"github.com/jaanek/hassext/snapcast"
	"github.com/zerodha/logf"
)

const (
	TopicLeiliruum    = "sound-switch-leiliruum"
	TopicSaunaEesruum = "sound-switch-sauna-eesruum"
)

type Sound interface {
	Init()
	Shutdown()
}

type SoundSwitch struct {
	lo               logf.Logger
	enabled          bool
	snapcast         snapcast.Snapcast
	snapcastClientId string
}

func (s *SoundSwitch) Receive(data []byte) error {
	if !s.enabled {
		s.lo.Debug("Sound switch. On data receive. Switch disabled", "name", s.snapcastClientId)
		return nil
	}
	msg := &IkeaSwitchMessage{}
	err := json.Unmarshal(data, msg)
	if err != nil {
		return err
	}
	switch msg.Action {
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
		s.snapcast.SendRequest(snapcast.ChangeStreamReq{
			ClientId: s.snapcastClientId,
			Up:       true,
		})
	case "off":
		s.snapcast.SendRequest(snapcast.ChangeStreamReq{
			ClientId: s.snapcastClientId,
			Up:       false,
		})
	}
	return nil
}

type sound struct {
	hub          *hub.Hub
	leiliruum    *SoundSwitch
	saunaEesruum *SoundSwitch
}

func New(lo logf.Logger, hub *hub.Hub, snapcast snapcast.Snapcast) Sound {
	return &sound{
		hub: hub,
		leiliruum: &SoundSwitch{
			lo:               lo,
			enabled:          true,
			snapcast:         snapcast,
			snapcastClientId: "zone-leiliruum",
		},
		saunaEesruum: &SoundSwitch{
			lo:               lo,
			enabled:          false,
			snapcast:         snapcast,
			snapcastClientId: "",
		},
	}
}

func (s *sound) Init() {
	// register into the hub
	s.hub.Register <- hub.Registration{
		Topic:  TopicLeiliruum,
		Client: s.leiliruum,
	}
	s.hub.Register <- hub.Registration{
		Topic:  TopicSaunaEesruum,
		Client: s.saunaEesruum,
	}
}

func (s *sound) Shutdown() {
	// unregister from hub
	s.hub.Unregister <- hub.Registration{
		Topic:  TopicLeiliruum,
		Client: s.leiliruum,
	}
	s.hub.Unregister <- hub.Registration{
		Topic:  TopicSaunaEesruum,
		Client: s.saunaEesruum,
	}
}
