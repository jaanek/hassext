package sound

import (
	"encoding/json"
	"time"

	"github.com/jaanek/hassext/chromecast"
	"github.com/jaanek/hassext/hub"
	"github.com/jaanek/hassext/jblbar"
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
	const chromecastVolumeStart = 0.31

	// first check if client is muted, if so then first action behaves as unmute
	var unmuteIfMuted = func() (*snapcast.Client, bool, error) {
		client, err := s.snapcast.ClientGetStatus(s.snapcastClientId)
		if err != nil {
			return client, false, err
		}
		if client.Muted {
			if s.snapcastClientId == ClientIdElutubaTv {
				// set starting volume
				var cc = s.chromecasts.ChromecastByDeviceName(chromecast.LIVING_ROOM_JBL)
				if cc != nil {
					// power on first if it's off/stand-by
					var streamingResult = jblbar.GetStreamingStatusResult{}
					var err = cc.CallCommand(jblbar.CommandGetStreamingStatus, nil, &streamingResult)
					if err != nil {
						s.lo.Error(s.prefix+"jbl-bar streaming status failed", "error", err)
					} else if streamingResult.Status.Source == jblbar.StreamingSourceIdle {
						// power it on
						var cmdResult = jblbar.CommandResultError{}
						var powerOn = jblbar.KeyPressedPower
						err = cc.CallCommand(jblbar.CommandSendAppController, &powerOn, &cmdResult)
						if err != nil {
							s.lo.Error(s.prefix+"jbl-bar power on failed", "error", err)
						}
						// set to TV
						var setTv = jblbar.KeyPressedSourceTV
						err = cc.CallCommand(jblbar.CommandSendAppController, &setTv, &cmdResult)
						if err != nil {
							s.lo.Error(s.prefix+"jbl-bar set source to TV failed", "error", err)
						}
					} else if streamingResult.Status.Source != jblbar.StreamingSourceTV {
						// set to TV
						var cmdResult = jblbar.CommandResultError{}
						var setTv = jblbar.KeyPressedSourceTV
						err = cc.CallCommand(jblbar.CommandSendAppController, &setTv, &cmdResult)
						if err != nil {
							s.lo.Error(s.prefix+"jbl-bar set source to TV failed", "error", err)
						}
					}

					// set default volume for sound
					err = cc.SetVolume(chromecastVolumeStart)
					if err != nil {
						s.lo.Error(s.prefix+"Setting start volume failed", "error", err)
					}
				} else {
					s.lo.Warn(s.prefix+"Living room chromecast not found", "chromecast name", chromecast.LIVING_ROOM_JBL)
				}
				// if morning (6am-9am) then set skyplus
				now := time.Now()
				morningStart := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, now.Location())
				morningEnd := time.Date(now.Year(), now.Month(), now.Day(), 9, 30, 0, 0, now.Location())
				if now.After(morningStart) && now.Before(morningEnd) {
					var streamId = "SkyPlus"
					s.snapcast.SendRequest(snapcast.SetDefaultChannelReq{
						ClientId: s.snapcastClientId,
						StreamId: streamId,
					})
				}
			}
			return client, true, s.snapcast.ClientMute(client, false)
		}
		return client, false, nil
	}

	// check which action to take
	switch msg.Action {
	// remote new
	case "toggle": // in new update versions is this
		fallthrough
	case "play_pause":
		var client, wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		return s.snapcast.ClientMute(client, true)
	// remote old
	case "brightness_move_up":
		var _, wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		// "top" long press. Mute or unmute
		s.snapcast.SendRequest(snapcast.MuteOnOffReq{
			ClientId: s.snapcastClientId,
		})
	case "brightness_move_down":
		var _, wasMuted, err = unmuteIfMuted()
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
	case "volume_down_hold":
		fallthrough
	case "arrow_left_hold":
		var _, wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		// "left" long press. Set stream channel
		var streamId = ""
		switch s.snapcastClientId {
		case ClientIdElutubaTv:
			streamId = "BluetoothHome"
		}
		if streamId == "" {
			return nil
		}
		s.snapcast.SendRequest(snapcast.SetDefaultChannelReq{
			ClientId: s.snapcastClientId,
			StreamId: streamId,
		})
	case "volume_up_hold":
		fallthrough
	case "arrow_right_hold":
		var _, wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		// "right" long press. Set stream channel
		var streamId = ""
		switch s.snapcastClientId {
		case ClientIdElutubaTv:
			streamId = "SkyPlus"
		}
		if streamId == "" {
			return nil
		}
		s.snapcast.SendRequest(snapcast.SetDefaultChannelReq{
			ClientId: s.snapcastClientId,
			StreamId: streamId,
		})
	case "volume_up":
		fallthrough
	case "arrow_right_click":
		var _, wasMuted, err = unmuteIfMuted()
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
	case "volume_down":
		fallthrough
	case "arrow_left_click":
		var _, wasMuted, err = unmuteIfMuted()
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
	case "track_previous":
		fallthrough
	case "dots_1_initial_press":
		fallthrough
	case "on":
		var _, wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		// top single press
		s.snapcast.SendRequest(snapcast.ChangeStreamReq{
			ClientId: s.snapcastClientId,
			Up:       true,
		})
	case "track_next":
		fallthrough
	case "dots_2_initial_press":
		fallthrough
	case "off":
		var _, wasMuted, err = unmuteIfMuted()
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
