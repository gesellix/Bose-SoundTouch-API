package main

import (
	"context"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/deborahgu/soundcork/internal/datastore"
	"github.com/deborahgu/soundcork/internal/models"
	"github.com/gesellix/bose-soundtouch/pkg/discovery"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	ds *datastore.DataStore
}

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
	server := &Server{ds: ds}

	// Phase 5: Device Discovery
	go func() {
		for {
			log.Println("Scanning for Bose devices...")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			svc := discovery.NewService(10 * time.Second)
			devices, err := svc.DiscoverDevices(ctx)
			cancel()
			if err != nil {
				log.Printf("Discovery error: %v", err)
			} else {
				for _, d := range devices {
					log.Printf("Discovered Bose device: %s at %s", d.Name, d.Host)
					// Update datastore for a default account (e.g., "default")
					// In a real scenario, we might need to know which account this device belongs to.
					info := &models.DeviceInfo{
						DeviceID:           d.SerialNo,
						Name:               d.Name,
						IPAddress:          d.Host,
						DeviceSerialNumber: d.SerialNo,
						ProductCode:        d.ModelID,
						FirmwareVersion:    "0.0.0", // Unknown from discovery
					}
					if err := ds.SaveDeviceInfo("default", d.SerialNo, info); err != nil {
						log.Printf("Failed to save device info: %v", err)
					}
				}
			}
			time.Sleep(5 * time.Minute)
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Phase 2: Root endpoint implemented in Go
	r.Get("/", server.handleRoot)

	// Phase 2: Static file serving for /media
	// In the project root, media is in soundcork/media
	// But in Docker, we might need to be careful about the paths.
	// Let's assume the media is accessible at ./soundcork/media relative to where the binary runs.
	mediaDir := os.Getenv("MEDIA_DIR")
	if mediaDir == "" {
		mediaDir = "soundcork/media"
	}
	r.Get("/media/*", server.handleMedia(mediaDir))

	// Phase 3: BMX endpoints
	r.Route("/bmx", func(r chi.Router) {
		r.Get("/registry/v1/services", server.handleBMXRegistry)
		r.Get("/tunein/v1/playback/station/{stationID}", server.handleTuneInPlayback)
		r.Get("/tunein/v1/playback/episodes/{podcastID}", server.handleTuneInPodcastInfo)
		r.Get("/tunein/v1/playback/episode/{podcastID}", server.handleTuneInPlaybackPodcast)
		r.Post("/orion/v1/playback/station/{data}", server.handleOrionPlayback)
	})

	// Phase 4: Marge endpoints
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

	// Delegation Logic: Proxy everything else to Python
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying request: %s %s -> %s", r.Method, r.URL.Path, targetURL)
		proxy.ServeHTTP(w, r)
	})

	log.Printf("Go service starting on %s, proxying to %s", addr, targetURL)
	log.Fatal(http.ListenAndServe(addr, r))
}
