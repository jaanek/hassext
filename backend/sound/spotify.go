package sound

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jaanek/hassext/spotify"
	"github.com/zerodha/logf"
)

const (
	// how often the listening history is polled from Spotify (Spotify keeps
	// only the last 50 played tracks, i.e. a few hours)
	spotifyHistoryPollInterval = 15 * time.Minute
	// how often the library playlists are merged into the history
	spotifyPlaylistsPollInterval = 6 * time.Hour
	// the channel list used for cycling is re-taken from the history when it
	// is older than this AND no channel was selected within this time (so
	// that the order stays stable while the user is cycling)
	spotifyChannelsMaxAge = 10 * time.Minute
	// max number of contexts kept in the persisted history
	spotifyHistoryMax = 200

	spotifyHistoryFile = "spotify-channels.json"
)

// SpotifyRemote turns a Spotify Connect receiver (librespot on the living
// room pi) into a "radio" that is driven by the buttons of the IKEA remote:
//
//   - play/pause toggles Spotify playback on the device
//   - back/forward select a "channel": one of the playlists/albums/artists
//     the user has listened to (most recent first)
//
// Spotify's API only exposes the last 50 played tracks, so the remote keeps
// its own persistent listening history (<dataDir>/spotify-channels.json),
// polled in the background and seeded with the library playlists.
// SpotifyOptions configures the SpotifyRemote.
type SpotifyOptions struct {
	// DeviceName is the Spotify Connect name of the receiver (librespot).
	DeviceName string
	// DataDir is where the listening history is persisted.
	DataDir string
	// MorningPlaylist is the name of a library playlist (or a spotify:playlist:
	// uri) that the play button starts instead of resuming when pressed
	// between MorningFrom and MorningTo ("HH:MM", local time). Optional.
	MorningPlaylist string
	MorningFrom     string
	MorningTo       string
}

type SpotifyRemote struct {
	prefix     string
	lo         logf.Logger
	sp         spotify.Spotify
	deviceName string
	dataDir    string
	morning    SpotifyOptions
	morningUri string // resolved uri of the morning playlist

	seq atomic.Int64 // bumped on every user command; stops stale verifications
	// lastSkipMs throttles song skips: too many track changes in a short
	// window trip Spotify's audio-key rate limit, after which librespot marks
	// tracks unavailable and races through the whole queue on its own
	lastSkipMs atomic.Int64

	mu         sync.Mutex
	device     *spotify.Device   // cached: the device id is stable while librespot runs
	history    []spotify.Context // persisted listening history, most recently played first
	contexts   []spotify.Context // channel list snapshot used for cycling
	cursor     int               // index in contexts of the selected channel, -1 if none
	current    string            // uri of the selected/playing context
	fetchedAt  time.Time         // when contexts was taken from the history
	selectedAt time.Time         // last channel selection
}

func NewSpotifyRemote(lo logf.Logger, sp spotify.Spotify, opts SpotifyOptions) *SpotifyRemote {
	r := &SpotifyRemote{
		prefix:     "[spotify-remote] ",
		lo:         lo,
		sp:         sp,
		deviceName: opts.DeviceName,
		dataDir:    opts.DataDir,
		morning:    opts,
		cursor:     -1,
	}
	if opts.MorningPlaylist != "" {
		if _, _, err := r.morningWindow(); err != nil {
			lo.Warn(r.prefix+"Invalid morning window, morning playlist disabled", "from", opts.MorningFrom, "to", opts.MorningTo, "error", err)
			r.morning.MorningPlaylist = ""
		} else {
			lo.Info(r.prefix+"Morning playlist configured", "playlist", opts.MorningPlaylist, "from", opts.MorningFrom, "to", opts.MorningTo)
		}
	}
	r.loadHistory()
	return r
}

// morningWindow parses the configured "HH:MM" window.
func (r *SpotifyRemote) morningWindow() (from, to time.Duration, err error) {
	parse := func(s string) (time.Duration, error) {
		t, err := time.Parse("15:04", strings.TrimSpace(s))
		if err != nil {
			return 0, err
		}
		return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute, nil
	}
	if from, err = parse(r.morning.MorningFrom); err != nil {
		return
	}
	if to, err = parse(r.morning.MorningTo); err != nil {
		return
	}
	if to <= from {
		err = errors.New("morning end must be after start")
	}
	return
}

// isMorning reports whether now is inside the morning window.
func (r *SpotifyRemote) isMorning(now time.Time) bool {
	if r.morning.MorningPlaylist == "" {
		return false
	}
	from, to, err := r.morningWindow()
	if err != nil {
		return false
	}
	sinceMidnight := time.Duration(now.Hour())*time.Hour + time.Duration(now.Minute())*time.Minute + time.Duration(now.Second())*time.Second
	return sinceMidnight >= from && sinceMidnight < to
}

// resolveMorningPlaylist finds the configured morning playlist: a spotify uri
// is used as is, a name is looked up in the listening history (which holds
// the library playlists) and, failing that, in the library directly.
func (r *SpotifyRemote) resolveMorningPlaylist() string {
	name := strings.TrimSpace(r.morning.MorningPlaylist)
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "spotify:") {
		return name
	}
	r.mu.Lock()
	uri := r.morningUri
	if uri == "" {
		for _, ctx := range r.history {
			if strings.EqualFold(ctx.Hint, name) && (ctx.Type == "" || ctx.Type == "playlist") {
				uri = ctx.Uri
				break
			}
		}
	}
	r.mu.Unlock()
	if uri == "" {
		playlists, err := r.sp.UserPlaylists()
		if err != nil {
			r.lo.Warn(r.prefix+"Looking up morning playlist in the library failed", "playlist", name, "error", err)
			return ""
		}
		r.mergeHistory(playlists)
		for _, p := range playlists {
			if strings.EqualFold(p.Hint, name) {
				uri = p.Uri
				break
			}
		}
	}
	if uri == "" {
		r.lo.Warn(r.prefix+"Morning playlist not found in the library (follow/save it in Spotify)", "playlist", name)
		return ""
	}
	r.mu.Lock()
	r.morningUri = uri
	r.mu.Unlock()
	return uri
}

// Run polls the listening history in the background until ctx is done.
func (r *SpotifyRemote) Run(ctx context.Context) {
	playlistsAt := time.Time{}
	poll := func() {
		if err := r.pollHistory(); err != nil {
			r.lo.Warn(r.prefix+"Polling listening history failed", "error", err)
		}
		if time.Since(playlistsAt) > spotifyPlaylistsPollInterval {
			if err := r.pollPlaylists(); err != nil {
				r.lo.Warn(r.prefix+"Fetching library playlists failed (re-run cmd/spotify-auth if the scope is missing)", "error", err)
			}
			playlistsAt = time.Now()
		}
	}
	// give the rest of the system a moment to start
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		return
	}
	poll()
	ticker := time.NewTicker(spotifyHistoryPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			poll()
		case <-ctx.Done():
			return
		}
	}
}

func (r *SpotifyRemote) isOurDevice(d spotify.Device) bool {
	return strings.EqualFold(d.Name, r.deviceName)
}

// playingHere reports whether Spotify is playing on our device right now.
func (r *SpotifyRemote) playingHere(state *spotify.PlayerState) bool {
	return state != nil && state.IsPlaying && r.isOurDevice(state.Device)
}

// withDevice runs fn with our device, looking it up (one api call) only when
// it is not cached yet. If Spotify reports the device as gone (404) the cache
// is dropped, the device looked up again and fn retried once.
func (r *SpotifyRemote) withDevice(fn func(dev *spotify.Device) error) error {
	r.mu.Lock()
	dev := r.device
	r.mu.Unlock()
	var lastErr error
	for attempt := 0; attempt < 4; attempt++ {
		if dev == nil {
			var err error
			dev, err = r.sp.FindDevice(r.deviceName)
			if err != nil {
				// librespot crashes now and then and re-registers within a few
				// seconds of restarting: wait for it instead of failing the press
				if errors.Is(err, spotify.ErrNoDevice) && attempt < 3 {
					r.lo.Warn(r.prefix+"Device not in the list (receiver restarting?), waiting", "device", r.deviceName, "attempt", attempt)
					lastErr = err
					time.Sleep(2500 * time.Millisecond)
					continue
				}
				return err
			}
			r.mu.Lock()
			r.device = dev
			r.mu.Unlock()
		}
		err := fn(dev)
		if err == nil || !spotify.IsNotFound(err) {
			return err
		}
		r.lo.Warn(r.prefix+"Device not found, looking it up again", "device", dev.Name, "error", err)
		lastErr = err
		r.mu.Lock()
		r.device = nil
		r.mu.Unlock()
		dev = nil
	}
	return lastErr
}

// prepare runs beforeStart concurrently and returns a channel that is closed
// when it is done.
func (r *SpotifyRemote) prepare(beforeStart func()) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		if beforeStart != nil {
			beforeStart()
		}
	}()
	return done
}

// Toggle pauses Spotify if it is playing on our device, otherwise starts it
// there. beforeStart is called (when not nil) while playback is being
// started, so that the caller can prepare the audio output. Returns whether
// playback was started.
func (r *SpotifyRemote) Toggle(beforeStart func()) (bool, error) {
	r.bumpSeq()
	state, err := r.sp.Player()
	if err != nil {
		return false, err
	}
	if r.playingHere(state) {
		err := r.sp.Pause(state.Device.Id)
		r.lo.Info(r.prefix+"Paused", "device", state.Device.Name, "playing", r.describe(state))
		return false, err
	}
	// fire and forget: preparing the output (sound bar power on etc) can be
	// slow and must never block the button handling
	r.prepare(beforeStart)
	return true, r.start(state)
}

// start starts playback on our device: resumes the current playback (wherever
// it is) when there is something to resume, otherwise plays the most recently
// listened channel.
func (r *SpotifyRemote) start(state *spotify.PlayerState) error {
	return r.withDevice(func(dev *spotify.Device) error {
		// in the morning the play button starts the morning playlist, unless
		// that is what is paused right now (then just resume it)
		if r.isMorning(time.Now()) {
			if uri := r.resolveMorningPlaylist(); uri != "" && !(state != nil && state.Context != nil && state.Context.Uri == uri) {
				if err := r.sp.Play(dev.Id, uri); err != nil {
					if spotify.IsNotFound(err) {
						return err
					}
					r.lo.Warn(r.prefix+"Playing morning playlist failed, falling back", "playlist", r.morning.MorningPlaylist, "error", err)
				} else {
					r.setCurrent(uri)
					r.verifyStarted(uri)
					r.lo.Info(r.prefix+"Playing morning playlist", "device", dev.Name, "playlist", r.morning.MorningPlaylist, "uri", uri)
					return nil
				}
			}
		}
		if state != nil && (state.Context != nil || state.Item != nil) {
			var err error
			if r.isOurDevice(state.Device) {
				err = r.sp.Play(dev.Id, "")
			} else {
				err = r.sp.TransferPlayback(dev.Id, true)
			}
			if err == nil {
				if state.Context != nil {
					r.setCurrent(state.Context.Uri)
				}
				r.verifyStarted("")
				r.lo.Info(r.prefix+"Resumed", "device", dev.Name, "from", state.Device.Name, "playing", r.describe(state))
				return nil
			}
			if spotify.IsNotFound(err) {
				return err
			}
			r.lo.Warn(r.prefix+"Resuming failed, playing most recent channel instead", "error", err)
		}

		r.mu.Lock()
		defer r.mu.Unlock()
		if err := r.refreshChannels(state); err != nil {
			return err
		}
		if len(r.contexts) == 0 {
			return errors.New("nothing to play: no listening history yet")
		}
		return r.playChannel(dev, 0)
	})
}

// Select plays the previous (forward=false, i.e. older) or the next
// (forward=true, i.e. more recent) channel of the listening history on our
// device. beforeStart is called when Spotify was not playing on our device
// yet. Returns the selected channel.
func (r *SpotifyRemote) Select(forward bool, beforeStart func()) (spotify.Context, error) {
	r.bumpSeq()
	var selected spotify.Context
	state, err := r.sp.Player()
	if err != nil {
		return selected, err
	}
	if !r.playingHere(state) {
		// fire and forget: prepare the audio output while the channel starts
		r.prepare(beforeStart)
	}
	err = r.withDevice(func(dev *spotify.Device) error {
		r.mu.Lock()
		defer r.mu.Unlock()
		if err := r.refreshChannels(state); err != nil {
			return err
		}
		n := len(r.contexts)
		if n == 0 {
			return errors.New("no listening history yet")
		}
		// nothing selected yet: start from the most recent one
		if r.cursor < 0 {
			if err := r.playChannel(dev, 0); err != nil {
				return err
			}
			selected = r.contexts[0]
			return nil
		}
		step := 1 // back: older
		if forward {
			step = -1 // forward: more recent
		}
		var lastErr error
		idx := r.cursor
		for i := 0; i < n-1; i++ {
			idx = ((idx+step)%n + n) % n
			if err := r.playChannel(dev, idx); err != nil {
				if spotify.IsNotFound(err) {
					return err
				}
				// e.g. a context type that can not be played on this device. Skip it.
				r.lo.Warn(r.prefix+"Playing channel failed, skipping", "uri", r.contexts[idx].Uri, "error", err)
				lastErr = err
				continue
			}
			selected = r.contexts[idx]
			return nil
		}
		if lastErr == nil {
			lastErr = errors.New("no other channel to select")
		}
		return lastErr
	})
	return selected, err
}

// Skip goes to the next (forward=true) or previous song of the playing
// channel. Returns false, without doing anything, when Spotify is not
// playing on our device - the buttons then keep their radio meaning.
func (r *SpotifyRemote) Skip(forward bool) (bool, error) {
	r.bumpSeq()
	state, err := r.sp.Player()
	if err != nil {
		return false, err
	}
	if !r.playingHere(state) {
		return false, nil
	}
	now := time.Now().UnixMilli()
	if now-r.lastSkipMs.Load() < 1200 {
		r.lo.Info(r.prefix+"Skip ignored, too soon after the previous one", "forward", forward)
		return true, nil
	}
	r.lastSkipMs.Store(now)
	if forward {
		err = r.sp.Next(state.Device.Id)
	} else {
		// note: librespot treats "previous" like the Spotify app does — more
		// than ~3s into a song it rewinds to the start, and only a quick
		// second press jumps to the previous song
		err = r.sp.Previous(state.Device.Id)
	}
	if err == nil {
		r.lo.Info(r.prefix+"Skipped", "forward", forward, "from", r.describe(state))
	}
	return true, err
}

// Stop pauses Spotify when it is playing on our device. Returns whether it
// was playing.
func (r *SpotifyRemote) Stop() (bool, error) {
	r.bumpSeq()
	state, err := r.sp.Player()
	if err != nil {
		return false, err
	}
	if !r.playingHere(state) {
		return false, nil
	}
	err = r.sp.Pause(state.Device.Id)
	r.lo.Info(r.prefix+"Stopped", "device", state.Device.Name, "playing", r.describe(state))
	return true, err
}

func (r *SpotifyRemote) bumpSeq() int64 { return r.seq.Add(1) }
func (r *SpotifyRemote) curSeq() int64  { return r.seq.Load() }

// verifyStarted checks in the background that playback really started on our
// device after a successful play command, and retries it when it did not.
// A play command is accepted by Spotify even when the receiver (librespot)
// has silently lost its session; Spotify then drops the device and librespot
// re-registers within ~20s - so retry with a fresh device lookup. The retries
// stop as soon as the user issues a newer command (seq changes).
func (r *SpotifyRemote) verifyStarted(uri string) {
	seq := r.curSeq()
	go func() {
		for _, delay := range []time.Duration{2500 * time.Millisecond, 4 * time.Second, 8 * time.Second, 8 * time.Second} {
			time.Sleep(delay)
			if r.curSeq() != seq {
				return // superseded by a newer command
			}
			state, err := r.sp.Player()
			if err != nil {
				r.lo.Warn(r.prefix+"Playback verification failed", "error", err)
				return
			}
			if r.playingHere(state) {
				return // all good
			}
			r.lo.Warn(r.prefix+"Playback did not start, retrying with a fresh device lookup", "uri", uri)
			r.mu.Lock()
			r.device = nil
			r.mu.Unlock()
			err = r.withDevice(func(dev *spotify.Device) error {
				return r.sp.Play(dev.Id, uri)
			})
			if err != nil {
				r.lo.Warn(r.prefix+"Playback retry failed", "uri", uri, "error", err)
				continue
			}
			r.lo.Info(r.prefix+"Playback retried", "uri", uri)
		}
	}()
}

// playChannel plays contexts[idx] on the device. Must be called with the lock held.
func (r *SpotifyRemote) playChannel(dev *spotify.Device, idx int) error {
	ctx := r.contexts[idx]
	if err := r.sp.Play(dev.Id, ctx.Uri); err != nil {
		return err
	}
	r.cursor = idx
	r.current = ctx.Uri
	r.selectedAt = time.Now()
	r.verifyStarted(ctx.Uri)
	total := len(r.contexts)
	// resolving the name may need an api call: keep it off the button path
	go r.lo.Info(r.prefix+"Playing channel", "device", dev.Name, "channel", r.channelName(ctx), "index", idx, "of", total, "uri", ctx.Uri)
	return nil
}

func (r *SpotifyRemote) setCurrent(uri string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.current = uri
	r.cursor = indexOf(r.contexts, uri)
}

func indexOf(contexts []spotify.Context, uri string) int {
	for i, ctx := range contexts {
		if ctx.Uri == uri {
			return i
		}
	}
	return -1
}

// refreshChannels (re)takes the channel list from the listening history when
// it is stale (and the user is not in the middle of cycling), and aligns the
// cursor with what is currently playing on our device (the user may have
// selected something from the phone meanwhile). Must be called with the lock
// held.
func (r *SpotifyRemote) refreshChannels(state *spotify.PlayerState) error {
	if state != nil && state.Context != nil && r.isOurDevice(state.Device) {
		r.current = state.Context.Uri
	}
	stale := time.Since(r.fetchedAt) > spotifyChannelsMaxAge && time.Since(r.selectedAt) > spotifyChannelsMaxAge
	if len(r.contexts) == 0 || stale {
		// pick up what was listened to since the last poll (best effort)
		r.mu.Unlock()
		err := r.pollHistory()
		r.mu.Lock()
		if err != nil {
			r.lo.Warn(r.prefix+"Fetching recently played failed, using stored history", "error", err)
		}
		r.contexts = append([]spotify.Context(nil), r.history...)
		r.fetchedAt = time.Now()
		r.lo.Info(r.prefix+"Channel list refreshed", "count", len(r.contexts))
	}
	if len(r.contexts) == 0 {
		return errors.New("no listening history yet")
	}
	if r.current == "" {
		r.cursor = -1
		return nil
	}
	r.cursor = indexOf(r.contexts, r.current)
	if r.cursor < 0 {
		// currently playing context is not in the history yet: put it in front
		r.contexts = append([]spotify.Context{{Uri: r.current}}, r.contexts...)
		r.cursor = 0
	}
	return nil
}

// ---------------------------------------------------------------------------
// Listening history

func (r *SpotifyRemote) historyFile() string {
	if r.dataDir == "" {
		return ""
	}
	return filepath.Join(r.dataDir, spotifyHistoryFile)
}

func (r *SpotifyRemote) loadHistory() {
	file := r.historyFile()
	if file == "" {
		return
	}
	body, err := os.ReadFile(file)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			r.lo.Warn(r.prefix+"Reading listening history failed", "file", file, "error", err)
		}
		return
	}
	var history []spotify.Context
	if err := json.Unmarshal(body, &history); err != nil {
		r.lo.Warn(r.prefix+"Parsing listening history failed", "file", file, "error", err)
		return
	}
	r.mu.Lock()
	r.history = history
	r.mu.Unlock()
	r.lo.Info(r.prefix+"Listening history loaded", "file", file, "count", len(history))
}

// saveHistory writes the history to disk. Must be called with the lock held.
func (r *SpotifyRemote) saveHistory() {
	file := r.historyFile()
	if file == "" {
		return
	}
	body, err := json.MarshalIndent(r.history, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(file, body, 0644); err != nil {
		r.lo.Warn(r.prefix+"Writing listening history failed", "file", file, "error", err)
	}
}

// mergeHistory merges contexts into the history: the last played time is
// kept at its maximum, the history is sorted most recently played first
// (never played, e.g. library playlists, last) and capped.
func (r *SpotifyRemote) mergeHistory(contexts []spotify.Context) (changed bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ctx := range contexts {
		if ctx.Uri == "" {
			continue
		}
		idx := indexOf(r.history, ctx.Uri)
		if idx < 0 {
			r.history = append(r.history, ctx)
			changed = true
			continue
		}
		entry := &r.history[idx]
		if ctx.PlayedAt.After(entry.PlayedAt) {
			entry.PlayedAt = ctx.PlayedAt
			changed = true
		}
		if ctx.Hint != "" && entry.Hint != ctx.Hint {
			entry.Hint = ctx.Hint
			changed = true
		}
		if ctx.Type != "" && entry.Type == "" {
			entry.Type = ctx.Type
			changed = true
		}
	}
	if !changed {
		return false
	}
	sort.SliceStable(r.history, func(i, j int) bool {
		return r.history[i].PlayedAt.After(r.history[j].PlayedAt)
	})
	if len(r.history) > spotifyHistoryMax {
		r.history = r.history[:spotifyHistoryMax]
	}
	r.saveHistory()
	return true
}

// pollHistory merges Spotify's recently played contexts into the history.
func (r *SpotifyRemote) pollHistory() error {
	contexts, err := r.sp.RecentlyPlayedContexts()
	if err != nil {
		return err
	}
	if r.mergeHistory(contexts) {
		r.mu.Lock()
		n := len(r.history)
		r.mu.Unlock()
		r.lo.Info(r.prefix+"Listening history updated", "recent", len(contexts), "total", n)
	}
	return nil
}

// pollPlaylists merges the library playlists into the history (as never
// played, unless already known).
func (r *SpotifyRemote) pollPlaylists() error {
	playlists, err := r.sp.UserPlaylists()
	if err != nil {
		return err
	}
	if r.mergeHistory(playlists) {
		r.mu.Lock()
		n := len(r.history)
		r.mu.Unlock()
		r.lo.Info(r.prefix+"Library playlists merged into history", "playlists", len(playlists), "total", n)
	}
	return nil
}

// Channels returns the listening history, most recently played first.
func (r *SpotifyRemote) Channels() []spotify.Context {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]spotify.Context(nil), r.history...)
}

func (r *SpotifyRemote) channelName(ctx spotify.Context) string {
	name, err := r.sp.ContextName(ctx.Uri)
	if err != nil || name == "" {
		if ctx.Hint != "" {
			return fmt.Sprintf("%s (%s)", ctx.Uri, ctx.Hint)
		}
		return ctx.Uri
	}
	return name
}

func (r *SpotifyRemote) describe(state *spotify.PlayerState) string {
	if state == nil {
		return "-"
	}
	var parts []string
	if state.Item != nil {
		parts = append(parts, state.Item.Name)
	}
	if state.Context != nil {
		parts = append(parts, "from "+r.channelName(*state.Context))
	}
	return strings.Join(parts, " ")
}

// ---------------------------------------------------------------------------
// IkeaButtons glue

// spotifyToggle handles the play/pause button when the zone drives Spotify.
func (s *IkeaButtons) spotifyToggle(ensureOnline func()) error {
	_, err := s.spotify.Toggle(func() { s.spotifyOutputOn(ensureOnline) })
	if err != nil {
		s.lo.Error(s.prefix+"Spotify play/pause failed", "error", err)
		return err
	}
	return nil
}

// spotifySelect handles the back/forward buttons: selects a channel of the
// listening history (playlist/album/artist ...).
func (s *IkeaButtons) spotifySelect(forward bool, ensureOnline func()) error {
	ctx, err := s.spotify.Select(forward, func() { s.spotifyOutputOn(ensureOnline) })
	if err != nil {
		s.lo.Error(s.prefix+"Spotify channel select failed", "forward", forward, "error", err)
		return err
	}
	s.lo.Info(s.prefix+"Spotify channel selected", "uri", ctx.Uri, "hint", ctx.Hint, "forward", forward)
	return nil
}

// spotifySkip handles the dots buttons while Spotify plays on this zone:
// next (forward=true) / previous song of the playing channel. Returns whether
// the press was handled (false: Spotify not playing, keep the radio meaning).
func (s *IkeaButtons) spotifySkip(forward bool, ensureOnline func()) {
	handled, err := s.spotify.Skip(forward)
	if err != nil {
		s.lo.Error(s.prefix+"Spotify skip failed", "forward", forward, "error", err)
		return
	}
	if !handled {
		// nothing is playing (e.g. "previous" on the first track stopped the
		// player, or the receiver restarted): start playback again
		s.lo.Info(s.prefix+"Spotify skip: not playing, starting playback", "forward", forward)
		_ = s.spotifyToggle(ensureOnline)
	}
}

// spotifyOutputOn prepares the zone for Spotify: powers on the sound bar
// (ensureOnline) and mutes the snapcast client so that the radio does not
// play over Spotify. Runs concurrently with the Spotify calls.
func (s *IkeaButtons) spotifyOutputOn(ensureOnline func()) {
	s.outputMu.Lock()
	defer s.outputMu.Unlock()
	ensureOnline()
	client, err := s.snapcast.ClientGetStatus(s.snapcastClientId)
	if err != nil {
		s.lo.Warn(s.prefix+"Snapcast client status failed", "error", err)
		return
	}
	if !client.Muted {
		if err := s.snapcast.ClientMute(client, true); err != nil {
			s.lo.Warn(s.prefix+"Muting snapcast client failed", "error", err)
		}
	}
}
