package sound

import (
	"encoding/json"

	"github.com/jaanek/hassext/chromecast"
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
	prefix           string
	lo               logf.Logger
	snapcast         snapcast.Snapcast
	chromecasts      chromecast.Chromecasts
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
	s.lo.Info(s.prefix+"Message received", "msg", msg)
	const chromecastVolumeStep = 0.03999999910593033 // got from: "stepInterval\":0.03999999910593033
	const chromecastVolumeStart = 0.35

	// first check if client is muted, if so then first action behaves as unmute
	var unmuteIfMuted = func() (bool, error) {
		client, err := s.snapcast.ClientGetStatus(s.snapcastClientId)
		if err != nil {
			return false, err
		}
		if client.Muted {
			if s.snapcastClientId == ClientIdElutubaTv {
				var cc = s.chromecasts.ChromecastByDeviceName(chromecast.LIVING_ROOM_JBL)
				if cc != nil {
					var err = cc.SetVolume(chromecastVolumeStart)
					if err != nil {
						s.lo.Error(s.prefix+"Setting start volume failed", "error", err)
					}
				} else {
					s.lo.Warn(s.prefix+"Living room chromecast not found", "chromecast name", chromecast.LIVING_ROOM_JBL)
				}
			}
			return true, s.snapcast.ClientMute(client, false)
		}
		return false, nil
	}

	// check which action to take
	switch msg.Action {
	case "brightness_move_up":
		var wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		// "top" long press. Mute or unmute
		s.snapcast.SendRequest(snapcast.MuteOnOffReq{
			ClientId: s.snapcastClientId,
		})
	case "brightness_move_down":
		var wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
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
		var wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		switch s.snapcastClientId {
		case ClientIdElutubaTv:
			var cc = s.chromecasts.ChromecastByDeviceName(chromecast.LIVING_ROOM_JBL)
			if cc != nil {
				var err = cc.IncVolume(chromecastVolumeStep)
				if err != nil {
					s.lo.Error(s.prefix+"Increasing volume failed", "error", err)
				}
			} else {
				s.lo.Warn(s.prefix+"Living room chromecast not found", "chromecast name", chromecast.LIVING_ROOM_JBL)
			}
		default:
			s.snapcast.SendRequest(snapcast.IncVolumeReq{
				ClientId: s.snapcastClientId,
				IncStep:  2,
			})
		}
	case "arrow_left_click":
		var wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		switch s.snapcastClientId {
		case ClientIdElutubaTv:
			var cc = s.chromecasts.ChromecastByDeviceName(chromecast.LIVING_ROOM_JBL)
			if cc != nil {
				var err = cc.IncVolume(-chromecastVolumeStep)
				if err != nil {
					s.lo.Error(s.prefix+"Decreasing volume failed", "error", err)
				}
			} else {
				s.lo.Warn(s.prefix+"Living room chromecast not found", "chromecast name", chromecast.LIVING_ROOM_JBL)
			}
		default:
			s.snapcast.SendRequest(snapcast.IncVolumeReq{
				ClientId: s.snapcastClientId,
				IncStep:  -2,
			})
		}
	case "on":
		var wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		// top single press
		s.snapcast.SendRequest(snapcast.ChangeStreamReq{
			ClientId: s.snapcastClientId,
			Up:       true,
		})
	case "off":
		var wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
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

func New(lo logf.Logger, h *hub.Hub, snapcast snapcast.Snapcast, chromecasts chromecast.Chromecasts) Sound {
	return &sound{
		hub: h,
		listeners: []SoundListener{
			&IkeaButtons{
				prefix:           "[sound-ikea-buttons-leiliruum] ",
				lo:               lo,
				snapcast:         snapcast,
				chromecasts:      chromecasts,
				enabled:          true,
				listenTopic:      TopicLeiliruumSoundButtons,
				snapcastClientId: ClientIdLeiliruum,
			},
			&IkeaButtons{
				prefix:           "[sound-ikea-buttons-sauna-eesruum] ",
				lo:               lo,
				snapcast:         snapcast,
				chromecasts:      chromecasts,
				enabled:          true,
				listenTopic:      TopicSaunaEesruumSoundButtons,
				snapcastClientId: ClientIdSaunaEesruum,
			},
			&IkeaButtons{
				prefix:           "[sound-ikea-buttons-elutuba] ",
				lo:               lo,
				snapcast:         snapcast,
				chromecasts:      chromecasts,
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
