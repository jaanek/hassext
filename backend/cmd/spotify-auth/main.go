// spotify-auth performs the one-time Spotify OAuth2 authorization for hassext
// and prints the refresh token to put into config.toml ([spotify] section).
//
// Prerequisites: create an app at https://developer.spotify.com/dashboard
// with the redirect URI  http://127.0.0.1:8899/callback  (Web API enabled).
//
// Usage (run on a machine with a browser):
//
//	go run ./cmd/spotify-auth --client-id ... --client-secret ...
//	go run ./cmd/spotify-auth --config config.toml   # reads spotify.clientId/clientSecret
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jaanek/hassext/spotify"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
)

func main() {
	clientId := flag.String("client-id", "", "Spotify app client id")
	clientSecret := flag.String("client-secret", "", "Spotify app client secret")
	cfgPath := flag.String("config", "config.toml", "config file to read spotify.clientId/clientSecret from when flags are not given")
	port := flag.Int("port", 8899, "local port for the OAuth redirect (must match the app's redirect URI)")
	flag.Parse()

	if *clientId == "" || *clientSecret == "" {
		ko := koanf.New(".")
		if err := ko.Load(file.Provider(*cfgPath), toml.Parser()); err != nil {
			fail("client id/secret not given and config not readable: %v", err)
		}
		if *clientId == "" {
			*clientId = ko.String("spotify.clientId")
		}
		if *clientSecret == "" {
			*clientSecret = ko.String("spotify.clientSecret")
		}
	}
	if *clientId == "" || *clientSecret == "" {
		fail("spotify client id and secret are required")
	}

	redirectUri := fmt.Sprintf("http://127.0.0.1:%d/callback", *port)
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		fail("random state: %v", err)
	}
	state := hex.EncodeToString(stateBytes)

	q := url.Values{}
	q.Set("client_id", *clientId)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectUri)
	q.Set("scope", spotify.Scopes)
	q.Set("state", state)
	authUrl := spotify.AuthorizeUrl + "?" + q.Encode()

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", *port))
	if err != nil {
		fail("listen: %v", err)
	}
	fmt.Printf("\nMake sure the Spotify app has this redirect URI registered:\n  %s\n\n", redirectUri)
	fmt.Printf("Open this link in the browser and authorize:\n\n  %s\n\n", authUrl)

	done := make(chan struct{})
	client := &http.Client{Timeout: 15 * time.Second}
	server := &http.Server{}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			return
		}
		if e := r.URL.Query().Get("error"); e != "" {
			http.Error(w, "authorization failed: "+e, http.StatusBadRequest)
			fmt.Printf("authorization failed: %s\n", e)
			close(done)
			return
		}
		form := url.Values{}
		form.Set("grant_type", "authorization_code")
		form.Set("code", r.URL.Query().Get("code"))
		form.Set("redirect_uri", redirectUri)
		tok, err := spotify.RequestToken(client, *clientId, *clientSecret, form)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Printf("token exchange failed: %v\n", err)
			close(done)
			return
		}
		fmt.Fprintln(w, "<html><body><h2>hassext: Spotify authorized. You can close this window.</h2></body></html>")

		fmt.Printf("\nAuthorized. Put this into config.toml:\n\n[spotify]\nclientId = %q\nclientSecret = %q\nrefreshToken = %q\ndeviceName = \"LivingRoom\"\n\n", *clientId, *clientSecret, tok.RefreshToken)
		printDevices(client, tok.AccessToken)
		close(done)
	})
	go server.Serve(listener)
	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

// printDevices lists the Spotify Connect devices so the deviceName config
// value can be verified.
func printDevices(client *http.Client, accessToken string) {
	req, _ := http.NewRequest(http.MethodGet, spotify.ApiUrl+"/me/player/devices", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("listing devices failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	var result struct {
		Devices []spotify.Device `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("listing devices failed (%d): %v\n", resp.StatusCode, err)
		return
	}
	fmt.Println("Spotify Connect devices currently visible (use the name as deviceName):")
	for _, d := range result.Devices {
		fmt.Printf("  - %q (%s, active: %v)\n", d.Name, d.Type, d.IsActive)
	}
	if len(result.Devices) == 0 {
		fmt.Println("  (none - is librespot/raspotify running and logged in?)")
	}
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
