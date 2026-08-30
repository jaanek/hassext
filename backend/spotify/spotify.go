// Package spotify is a minimal Spotify Web API client used to drive a Spotify
// Connect receiver (librespot/raspotify) from physical buttons.
//
// It covers only what the remote needs: listing devices, reading the player
// state, transferring/starting/pausing playback and reading the user's
// recently played contexts (playlists, albums, artists, shows ...).
//
// Authentication uses the OAuth2 "authorization code" flow. The one-time
// authorization is done with `go run ./cmd/spotify-auth`, which prints the
// refresh token to put into config.toml. Access tokens are refreshed
// automatically; if Spotify hands out a rotated refresh token it is persisted
// into <dataDir>/spotify-token.json and takes precedence over the config.
package spotify

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zerodha/logf"
)

const (
	ApiUrl       = "https://api.spotify.com/v1"
	AccountsUrl  = "https://accounts.spotify.com"
	AuthorizeUrl = AccountsUrl + "/authorize"
	TokenUrl     = AccountsUrl + "/api/token"

	// Scopes needed by the remote: read/modify playback + recently played
	Scopes = "user-read-playback-state user-modify-playback-state user-read-recently-played playlist-read-private"

	tokenFileName = "spotify-token.json"
)

var ErrNoDevice = errors.New("spotify device not found")

type Spotify interface {
	// Devices lists the Spotify Connect devices visible to the user.
	Devices() ([]Device, error)
	// FindDevice returns the device with the given (case insensitive) name.
	FindDevice(name string) (*Device, error)
	// Player returns the current playback state, or nil when nothing is active.
	Player() (*PlayerState, error)
	// TransferPlayback moves the current playback to the given device.
	TransferPlayback(deviceId string, play bool) error
	// Play starts playing the context (playlist/album/artist/show uri) on the
	// device. Empty context resumes the current playback on that device.
	Play(deviceId string, contextUri string) error
	// Pause pauses playback on the device.
	Pause(deviceId string) error
	// RecentlyPlayedContexts returns the distinct contexts (playlists, albums,
	// ...) the user has recently listened to, most recent first. Spotify only
	// keeps the last 50 played tracks, so this covers a few hours at most.
	RecentlyPlayedContexts() ([]Context, error)
	// UserPlaylists returns the playlists in the user's library (own and
	// followed ones). Needs the playlist-read-private scope.
	UserPlaylists() ([]Context, error)
	// ContextName resolves a human readable name for a context uri. Names are
	// cached. Used for logging only - errors are returned, never fatal.
	ContextName(uri string) (string, error)
}

type Params struct {
	ClientId     string
	ClientSecret string
	RefreshToken string
	// DataDir is where a rotated refresh token is persisted. Optional.
	DataDir string
}

type Device struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	IsActive bool   `json:"is_active"`
	Volume   int    `json:"volume_percent"`
}

type Context struct {
	Type string `json:"type"`
	Uri  string `json:"uri"`
	// Hint is the name of a track that was played from this context, so that
	// logs are meaningful even without resolving the context name.
	Hint string `json:"hint,omitempty"`
	// PlayedAt is when the context was last played (zero if unknown).
	PlayedAt time.Time `json:"played_at,omitempty"`
}

type PlayerState struct {
	Device    Device   `json:"device"`
	IsPlaying bool     `json:"is_playing"`
	Context   *Context `json:"context"`
	Item      *struct {
		Name string `json:"name"`
		Uri  string `json:"uri"`
	} `json:"item"`
}

type ApiError struct {
	Status  int
	Message string
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("spotify api error %d: %s", e.Status, e.Message)
}

// IsNotFound reports whether the error is a 404 from Spotify (e.g. "Device
// not found" / "No active device").
func IsNotFound(err error) bool {
	var apiErr *ApiError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

type spotify struct {
	lo     logf.Logger
	params *Params
	http   *http.Client

	tokenMu      sync.Mutex
	accessToken  string
	tokenExpires time.Time
	refreshToken string

	namesMu sync.Mutex
	names   map[string]string
}

func New(lo logf.Logger, params *Params) Spotify {
	s := &spotify{
		lo:           lo,
		params:       params,
		http:         &http.Client{Timeout: 15 * time.Second},
		refreshToken: params.RefreshToken,
		names:        map[string]string{},
	}
	// a rotated refresh token stored earlier takes precedence over the config
	if stored := s.loadStoredToken(); stored != "" {
		s.refreshToken = stored
	}
	return s
}

// ---------------------------------------------------------------------------
// Token handling

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

func (s *spotify) tokenFile() string {
	if s.params.DataDir == "" {
		return ""
	}
	return filepath.Join(s.params.DataDir, tokenFileName)
}

func (s *spotify) loadStoredToken() string {
	file := s.tokenFile()
	if file == "" {
		return ""
	}
	body, err := os.ReadFile(file)
	if err != nil {
		return ""
	}
	var stored struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &stored); err != nil {
		s.lo.Warn("[spotify] stored token file unreadable", "file", file, "error", err)
		return ""
	}
	return stored.RefreshToken
}

func (s *spotify) storeToken(refreshToken string) {
	file := s.tokenFile()
	if file == "" {
		return
	}
	body, _ := json.Marshal(map[string]string{"refresh_token": refreshToken})
	if err := os.WriteFile(file, body, 0600); err != nil {
		s.lo.Warn("[spotify] storing rotated refresh token failed", "file", file, "error", err)
	}
}

// accessTokenValid returns a valid access token, refreshing it when needed.
func (s *spotify) accessTokenValid(force bool) (string, error) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	if !force && s.accessToken != "" && time.Now().Before(s.tokenExpires) {
		return s.accessToken, nil
	}
	if s.refreshToken == "" {
		return "", errors.New("spotify refresh token not configured (run: go run ./cmd/spotify-auth)")
	}
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", s.refreshToken)
	tok, err := RequestToken(s.http, s.params.ClientId, s.params.ClientSecret, form)
	if err != nil {
		return "", fmt.Errorf("spotify token refresh failed: %w", err)
	}
	s.accessToken = tok.AccessToken
	// refresh a bit early
	s.tokenExpires = time.Now().Add(time.Duration(tok.ExpiresIn)*time.Second - time.Minute)
	if tok.RefreshToken != "" && tok.RefreshToken != s.refreshToken {
		s.refreshToken = tok.RefreshToken
		s.storeToken(tok.RefreshToken)
	}
	s.lo.Info("[spotify] access token refreshed", "expires", s.tokenExpires.Format(time.RFC3339))
	return s.accessToken, nil
}

// RequestToken posts to the Spotify token endpoint with client credentials as
// basic auth. Shared with the cmd/spotify-auth tool.
func RequestToken(client *http.Client, clientId, clientSecret string, form url.Values) (*tokenResponse, error) {
	req, err := http.NewRequest(http.MethodPost, TokenUrl, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(clientId, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	tok := &tokenResponse{}
	if err := json.Unmarshal(body, tok); err != nil {
		return nil, fmt.Errorf("invalid token response (%d): %s", resp.StatusCode, string(body))
	}
	if resp.StatusCode != http.StatusOK || tok.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned %d: %s %s", resp.StatusCode, tok.Error, tok.ErrorDesc)
	}
	return tok, nil
}

// ---------------------------------------------------------------------------
// Low level api call

// call performs an authenticated request. A 401 triggers a single token
// refresh and retry. 2xx responses return the body (may be empty).
func (s *spotify) call(method, path string, query url.Values, payload any) ([]byte, error) {
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}
	u := ApiUrl + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	for attempt := 0; attempt < 2; attempt++ {
		token, err := s.accessTokenValid(attempt > 0)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(method, u, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := s.http.Do(req)
		if err != nil {
			return nil, err
		}
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			s.lo.Info("[spotify] unauthorized, refreshing access token", "path", path)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, &ApiError{Status: resp.StatusCode, Message: apiErrorMessage(respBody)}
		}
		s.lo.Debug("[spotify] api call", "method", method, "path", path, "status", resp.StatusCode)
		return respBody, nil
	}
	return nil, errors.New("spotify: unauthorized")
}

func apiErrorMessage(body []byte) string {
	var e struct {
		Error struct {
			Message string `json:"message"`
			Reason  string `json:"reason"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		if e.Error.Reason != "" {
			return e.Error.Message + " (" + e.Error.Reason + ")"
		}
		return e.Error.Message
	}
	return strings.TrimSpace(string(body))
}

// ---------------------------------------------------------------------------
// Player api

func (s *spotify) Devices() ([]Device, error) {
	body, err := s.call(http.MethodGet, "/me/player/devices", nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Devices []Device `json:"devices"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return result.Devices, nil
}

func (s *spotify) FindDevice(name string) (*Device, error) {
	devices, err := s.Devices()
	if err != nil {
		return nil, err
	}
	for i := range devices {
		if strings.EqualFold(devices[i].Name, name) {
			return &devices[i], nil
		}
	}
	names := make([]string, 0, len(devices))
	for _, d := range devices {
		names = append(names, d.Name)
	}
	return nil, fmt.Errorf("%w: %q (available: %s)", ErrNoDevice, name, strings.Join(names, ", "))
}

func (s *spotify) Player() (*PlayerState, error) {
	body, err := s.call(http.MethodGet, "/me/player", nil, nil)
	if err != nil {
		return nil, err
	}
	// 204 No Content: nothing is playing / no active device
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, nil
	}
	state := &PlayerState{}
	if err := json.Unmarshal(body, state); err != nil {
		return nil, err
	}
	return state, nil
}

func (s *spotify) TransferPlayback(deviceId string, play bool) error {
	_, err := s.call(http.MethodPut, "/me/player", nil, map[string]any{
		"device_ids": []string{deviceId},
		"play":       play,
	})
	return err
}

func (s *spotify) Play(deviceId string, contextUri string) error {
	q := url.Values{}
	q.Set("device_id", deviceId)
	var payload any
	if contextUri != "" {
		payload = map[string]any{"context_uri": contextUri}
	}
	_, err := s.call(http.MethodPut, "/me/player/play", q, payload)
	return err
}

func (s *spotify) Pause(deviceId string) error {
	q := url.Values{}
	q.Set("device_id", deviceId)
	_, err := s.call(http.MethodPut, "/me/player/pause", q, nil)
	return err
}

func (s *spotify) RecentlyPlayedContexts() ([]Context, error) {
	q := url.Values{}
	q.Set("limit", "50")
	body, err := s.call(http.MethodGet, "/me/player/recently-played", q, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Items []struct {
			Track struct {
				Name string `json:"name"`
			} `json:"track"`
			PlayedAt time.Time `json:"played_at"`
			Context  *Context  `json:"context"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	contexts := make([]Context, 0, len(result.Items))
	for _, item := range result.Items {
		if item.Context == nil || item.Context.Uri == "" || seen[item.Context.Uri] {
			continue
		}
		seen[item.Context.Uri] = true
		ctx := *item.Context
		ctx.Hint = item.Track.Name
		ctx.PlayedAt = item.PlayedAt
		contexts = append(contexts, ctx)
	}
	return contexts, nil
}

func (s *spotify) UserPlaylists() ([]Context, error) {
	var contexts []Context
	q := url.Values{}
	q.Set("limit", "50")
	for offset := 0; offset < 500; offset += 50 {
		q.Set("offset", fmt.Sprint(offset))
		body, err := s.call(http.MethodGet, "/me/playlists", q, nil)
		if err != nil {
			return nil, err
		}
		var result struct {
			Items []struct {
				Name string `json:"name"`
				Uri  string `json:"uri"`
			} `json:"items"`
			Next string `json:"next"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}
		for _, item := range result.Items {
			if item.Uri == "" {
				continue
			}
			contexts = append(contexts, Context{Type: "playlist", Uri: item.Uri, Hint: item.Name})
			s.namesMu.Lock()
			s.names[item.Uri] = item.Name
			s.namesMu.Unlock()
		}
		if result.Next == "" {
			break
		}
	}
	return contexts, nil
}

func (s *spotify) ContextName(uri string) (string, error) {
	s.namesMu.Lock()
	name, ok := s.names[uri]
	s.namesMu.Unlock()
	if ok {
		return name, nil
	}
	// uri format: spotify:<type>:<id>
	parts := strings.Split(uri, ":")
	if len(parts) < 3 {
		return "", fmt.Errorf("unsupported context uri: %s", uri)
	}
	var path string
	q := url.Values{}
	switch parts[1] {
	case "playlist":
		path = "/playlists/" + parts[2]
		q.Set("fields", "name")
	case "album":
		path = "/albums/" + parts[2]
	case "artist":
		path = "/artists/" + parts[2]
	case "show":
		path = "/shows/" + parts[2]
	case "user":
		if len(parts) == 4 && parts[3] == "collection" {
			name = "Liked Songs"
		}
	}
	if name == "" && path != "" {
		body, err := s.call(http.MethodGet, path, q, nil)
		if err != nil {
			return "", err
		}
		var result struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", err
		}
		name = result.Name
	}
	if name == "" {
		name = uri
	}
	s.namesMu.Lock()
	s.names[uri] = name
	s.namesMu.Unlock()
	return name, nil
}
