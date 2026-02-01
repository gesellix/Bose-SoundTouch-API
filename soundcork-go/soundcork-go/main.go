package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"github.com/deborahgu/soundcork/internal/bmx"
	"github.com/deborahgu/soundcork/internal/datastore"
	"github.com/deborahgu/soundcork/internal/marge"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	bindAddr := os.Getenv("BIND_ADDR")
	// If BIND_ADDR is explicitly set, use it. Otherwise, bind to all interfaces (IPv4 and IPv6).
	addr := bindAddr + ":" + port
	if bindAddr == "" {
		addr = ":" + port
	}

	targetURL := os.Getenv("PYTHON_BACKEND_URL")
	if targetURL == "" {
		targetURL = "http://localhost:8001"
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Failed to parse target URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}
	ds := datastore.NewDataStore(dataDir)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Phase 2: Root endpoint implemented in Go
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"Bose": "Can't Brick Us", "service": "Go/Chi"}`)
	})

	// Phase 2: Static file serving for /media
	// In the project root, media is in soundcork/media
	// But in Docker, we might need to be careful about the paths.
	// Let's assume the media is accessible at ./soundcork/media relative to where the binary runs.
	mediaDir := http.Dir(os.Getenv("MEDIA_DIR"))
	if os.Getenv("MEDIA_DIR") == "" {
		mediaDir = http.Dir("soundcork/media")
	}

	r.Get("/media/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/media/", http.FileServer(mediaDir))
		fs.ServeHTTP(w, r)
	})

	// Phase 3: BMX endpoints
	r.Route("/bmx", func(r chi.Router) {
		r.Get("/registry/v1/services", func(w http.ResponseWriter, r *http.Request) {
			// Read and process bmx_services.json
			data, err := os.ReadFile("soundcork/bmx_services.json")
			if err != nil {
				http.Error(w, "Failed to read services", http.StatusInternalServerError)
				return
			}

			baseURL := os.Getenv("BASE_URL")
			if baseURL == "" {
				baseURL = "http://localhost:8000" // Default for local dev
			}

			content := string(data)
			content = strings.ReplaceAll(content, "{BMX_SERVER}", baseURL)
			content = strings.ReplaceAll(content, "{MEDIA_SERVER}", baseURL+"/media")

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(content))
		})

		r.Get("/tunein/v1/playback/station/{stationID}", func(w http.ResponseWriter, r *http.Request) {
			stationID := chi.URLParam(r, "stationID")
			resp, err := bmx.TuneInPlayback(stationID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Get("/tunein/v1/playback/episodes/{podcastID}", func(w http.ResponseWriter, r *http.Request) {
			podcastID := chi.URLParam(r, "podcastID")
			encodedName := r.URL.Query().Get("encoded_name")
			resp, err := bmx.TuneInPodcastInfo(podcastID, encodedName)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Get("/tunein/v1/playback/episode/{podcastID}", func(w http.ResponseWriter, r *http.Request) {
			podcastID := chi.URLParam(r, "podcastID")
			resp, err := bmx.TuneInPlaybackPodcast(podcastID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})

		r.Post("/orion/v1/playback/station/{data}", func(w http.ResponseWriter, r *http.Request) {
			data := chi.URLParam(r, "data")
			resp, err := bmx.PlayCustomStream(data)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		})
	})

	// Phase 4: Marge endpoints
	r.Route("/marge", func(r chi.Router) {
		r.Get("/streaming/sourceproviders", func(w http.ResponseWriter, r *http.Request) {
			data, err := marge.SourceProvidersToXML()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.Write(data)
		})

		r.Get("/accounts/{account}/full", func(w http.ResponseWriter, r *http.Request) {
			account := chi.URLParam(r, "account")
			data, err := marge.AccountFullToXML(ds, account)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.Write(data)
		})

		r.Post("/streaming/support/power_on", func(w http.ResponseWriter, r *http.Request) {
			// Just return OK like the Python implementation
			w.WriteHeader(http.StatusOK)
		})

		r.Get("/updates/soundtouch", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(marge.SoftwareUpdateToXML()))
		})
	})

	// Delegation Logic: Proxy everything else to Python
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying request: %s %s -> %s", r.Method, r.URL.Path, targetURL)
		proxy.ServeHTTP(w, r)
	})

	log.Printf("Go service starting on %s, proxying to %s", addr, targetURL)
	log.Fatal(http.ListenAndServe(addr, r))
}
