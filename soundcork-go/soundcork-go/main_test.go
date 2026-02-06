package main

import (
	"net/http"
	"net/url"

	"github.com/deborahgu/soundcork/internal/datastore"
	"github.com/go-chi/chi/v5"
)

func setupRouter(targetURL string, ds *datastore.DataStore) (*chi.Mux, *Server) {
	target, _ := url.Parse(targetURL)
	proxy := &reverseProxy{target: target}
	server := &Server{ds: ds}

	r := chi.NewRouter()
	r.Get("/", server.handleRoot)

	// Setup media directory for tests
	mediaDir := "../../soundcork/media"
	r.Get("/media/*", server.handleMedia(mediaDir))

	// Setup BMX for tests
	r.Route("/bmx", func(r chi.Router) {
		r.Get("/registry/v1/services", server.handleBMXRegistry)
		r.Get("/tunein/v1/playback/station/{stationID}", server.handleTuneInPlayback)
		r.Get("/tunein/v1/playback/episodes/{podcastID}", server.handleTuneInPodcastInfo)
		r.Get("/tunein/v1/playback/episode/{podcastID}", server.handleTuneInPlaybackPodcast)
		r.Post("/orion/v1/playback/station/{data}", server.handleOrionPlayback)
	})

	// Setup Marge for tests
	r.Route("/marge", func(r chi.Router) {
		r.Get("/streaming/sourceproviders", server.handleMargeSourceProviders)
		r.Get("/accounts/{account}/full", server.handleMargeAccountFull)
		r.Post("/streaming/support/power_on", server.handleMargePowerOn)
		r.Get("/updates/soundtouch", server.handleMargeSoftwareUpdate)
		r.Get("/accounts/{account}/devices/{device}/presets", server.handleMargePresets)
		r.Post("/accounts/{account}/devices/{device}/presets/{presetNumber}", server.handleMargeUpdatePreset)
		r.Post("/accounts/{account}/devices/{device}/recents", server.handleMargeAddRecent)
		r.Post("/accounts/{account}/devices", server.handleMargeAddDevice)
		r.Delete("/accounts/{account}/devices/{device}", server.handleMargeRemoveDevice)
	})

	// Setup Setup for tests
	r.Route("/setup", func(r chi.Router) {
		r.Get("/proxy-settings", server.handleGetProxySettings)
		r.Post("/proxy-settings", server.handleUpdateProxySettings)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})
	return r, server
}

type reverseProxy struct {
	target *url.URL
}

func (p *reverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Simplified proxy for testing
	w.WriteHeader(http.StatusAccepted) // Custom status to identify proxy hit in tests
	w.Write([]byte("Proxied to " + p.target.String()))
}
