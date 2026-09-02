package sound

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/jaanek/hassext/chromecast"
	"github.com/jaanek/hassext/hub"
	"github.com/jaanek/hassext/jblbar"
	"github.com/jaanek/hassext/snapcast"
	"github.com/jaanek/hassext/spotify"
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
	// Run runs background work (Spotify listening history polling) until ctx is done.
	Run(context.Context)
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
	// optional: when set, the zone is a pure Spotify remote: play/pause toggles
	// Spotify, back/forward select a channel of the listening history, the dots
	// skip next/previous song. The snapcast (radio) stream actions are disabled
	// for the zone (the snapcast client stays muted).
	spotify  *SpotifyRemote
	outputMu sync.Mutex // serializes spotifyOutputOn (runs concurrently with Spotify calls)
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

	var ensureOnline = func() {
		if s.snapcastClientId == ClientIdElutubaTv {
			var cc = s.chromecasts.ChromecastByDeviceName(chromecast.LIVING_ROOM_JBL)
			if cc == nil {
				s.lo.Warn(s.prefix+"Living room chromecast not found", "chromecast name", chromecast.LIVING_ROOM_JBL)
				return
			}
			// power on first if it's off/stand-by
			var streamingResult = jblbar.GetStreamingStatusResult{}
			var err = cc.CallCommand(jblbar.CommandGetStreamingStatus, nil, &streamingResult)
			if err != nil {
				s.lo.Error(s.prefix+"jbl-bar streaming status failed", "error", err)
				return
			}
			var wasIdle = streamingResult.Status.Source == jblbar.StreamingSourceIdle
			if wasIdle {
				// power it on
				var cmdResult = jblbar.CommandResultError{}
				var powerOn = jblbar.KeyPressedPower
				err = cc.CallCommand(jblbar.CommandSendAppController, &powerOn, &cmdResult)
				if err != nil {
					s.lo.Error(s.prefix+"jbl-bar power on failed", "error", err)
				}
				// set to HDMI - this is just in case tv needs time when waking up ???
				var setHdmi = jblbar.KeyPressedSourceHDMI
				err = cc.CallCommand(jblbar.CommandSendAppController, &setHdmi, &cmdResult)
				if err != nil {
					s.lo.Error(s.prefix+"jbl-bar set source to HDMI failed", "error", err)
				}
				// switch to TV
				var setTv = jblbar.KeyPressedSourceTV
				err = cc.CallCommand(jblbar.CommandSendAppController, &setTv, &cmdResult)
				if err != nil {
					s.lo.Error(s.prefix+"jbl-bar set source to TV failed", "error", err)
				}

				// wait 1sec in a thread & check if is_streaming is true if not then  HDMI -> TV once again
				// {"error_code":"0","status":{"source":"TV","is_streaming":"false","is_atmos":"false"}}
				go func() {
					time.Sleep(1 * time.Second)
					var streamingResult = jblbar.GetStreamingStatusResult{}
					var err = cc.CallCommand(jblbar.CommandGetStreamingStatus, nil, &streamingResult)
					if err != nil {
						s.lo.Error(s.prefix+"jbl-bar streaming status failed", "error", err)
						return
					}
					if streamingResult.Status.IsStreaming == "false" {
						// set to HDMI - this is just in case tv needs time when waking up ???
						var setHdmi = jblbar.KeyPressedSourceHDMI
						err = cc.CallCommand(jblbar.CommandSendAppController, &setHdmi, &cmdResult)
						if err != nil {
							s.lo.Error(s.prefix+"jbl-bar set source to HDMI failed", "error", err)
						}
						// switch to TV
						var setTv = jblbar.KeyPressedSourceTV
						err = cc.CallCommand(jblbar.CommandSendAppController, &setTv, &cmdResult)
						if err != nil {
							s.lo.Error(s.prefix+"jbl-bar set source to TV failed", "error", err)
						}
					}
				}()
			} else if streamingResult.Status.Source != jblbar.StreamingSourceTV {
				// set to TV
				var cmdResult = jblbar.CommandResultError{}
				var setTv = jblbar.KeyPressedSourceTV
				err = cc.CallCommand(jblbar.CommandSendAppController, &setTv, &cmdResult)
				if err != nil {
					s.lo.Error(s.prefix+"jbl-bar set source to TV failed", "error", err)
				}
			} else if streamingResult.Status.IsStreaming == "false" {
				// at some point it's TV but not streaming
				// {"error_code":"0","status":{"source":"TV","is_streaming":"false","is_atmos":"false"}}
				// set to HDMI
				var cmdResult = jblbar.CommandResultError{}
				var setHdmi = jblbar.KeyPressedSourceHDMI
				err = cc.CallCommand(jblbar.CommandSendAppController, &setHdmi, &cmdResult)
				if err != nil {
					s.lo.Error(s.prefix+"jbl-bar set source to HDMI failed", "error", err)
				}
				// switch back to TV
				var setTv = jblbar.KeyPressedSourceTV
				err = cc.CallCommand(jblbar.CommandSendAppController, &setTv, &cmdResult)
				if err != nil {
					s.lo.Error(s.prefix+"jbl-bar set source to TV failed", "error", err)
				}
			}

			// set default volume only when waking the bar from idle; when it
			// was already on, keep whatever volume the user has set
			if wasIdle {
				err = cc.SetVolume(chromecastVolumeStart)
				if err != nil {
					s.lo.Error(s.prefix+"Setting start volume failed", "error", err)
				}
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
	}

	// first check if client is muted, if so then first action behaves as unmute
	var unmuteIfMuted = func() (*snapcast.Client, bool, error) {
		client, err := s.snapcast.ClientGetStatus(s.snapcastClientId)
		if err != nil {
			return client, false, err
		}
		if client.Muted {
			return client, true, s.snapcast.ClientMute(client, false)
		}
		return client, false, nil
	}

	// check which action to take
	switch msg.Action {
	// remote new
	case "toggle", "play_pause": // "toggle" in new zigbee2mqtt versions
		if s.spotify != nil {
			return s.spotifyToggle(ensureOnline)
		}
		var client, wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			if wasMuted {
				ensureOnline()
			}
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
	case "volume_down_hold", "arrow_left_hold":
		if s.spotify != nil {
			return nil // radio streams are disabled on the Spotify zone
		}
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
	case "volume_up_hold", "arrow_right_hold":
		if s.spotify != nil {
			return nil // radio streams are disabled on the Spotify zone
		}
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
	case "volume_up", "arrow_right_click":
		if s.spotify == nil { // never unmute the radio on the Spotify zone
			var _, wasMuted, err = unmuteIfMuted()
			if err != nil || wasMuted {
				return err
			}
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
	case "volume_down", "arrow_left_click":
		if s.spotify == nil { // never unmute the radio on the Spotify zone
			var _, wasMuted, err = unmuteIfMuted()
			if err != nil || wasMuted {
				return err
			}
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
	case "track_previous", "dots_1_initial_press", "on":
		if s.spotify != nil {
			if msg.Action == "dots_1_initial_press" {
				// single dot: previous song in the channel
				s.spotifySkip(false, ensureOnline)
				return nil
			}
			// back: older recently played Spotify channel
			return s.spotifySelect(false, ensureOnline)
		}
		var _, wasMuted, err = unmuteIfMuted()
		if err != nil || wasMuted {
			return err
		}
		// top single press
		s.snapcast.SendRequest(snapcast.ChangeStreamReq{
			ClientId: s.snapcastClientId,
			Up:       true,
		})
	case "track_next", "dots_2_initial_press", "off":
		if s.spotify != nil {
			if msg.Action == "dots_2_initial_press" {
				// double dot: next song in the channel
				s.spotifySkip(true, ensureOnline)
				return nil
			}
			// forward: more recent recently played Spotify channel
			return s.spotifySelect(true, ensureOnline)
		}
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
	spotify   *SpotifyRemote
}

// New creates the sound button listeners. sp is optional: when given, the
// living room (elutuba) remote drives Spotify on spotifyOpts.DeviceName.
func New(lo logf.Logger, h *hub.Hub, snapcast snapcast.Snapcast, chromecasts chromecast.Chromecasts, sp spotify.Spotify, spotifyOpts SpotifyOptions) Sound {
	var spotifyRemote *SpotifyRemote
	if sp != nil {
		spotifyRemote = NewSpotifyRemote(lo, sp, spotifyOpts)
	}
	return &sound{
		hub:     h,
		spotify: spotifyRemote,
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
				spotify:          spotifyRemote,
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

func (s *sound) Run(ctx context.Context) {
	if s.spotify != nil {
		s.spotify.Run(ctx)
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
