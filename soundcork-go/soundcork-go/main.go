package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/deborahgu/soundcork/internal/datastore"
	"github.com/deborahgu/soundcork/internal/models"
	"github.com/deborahgu/soundcork/internal/setup"
	"github.com/gesellix/bose-soundtouch/pkg/discovery"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	ds          *datastore.DataStore
	sm          *setup.Manager
	serverURL   string
	proxyURL    string
	discovering bool
}

func (s *Server) discoverDevices() {
	s.discovering = true
	defer func() { s.discovering = false }()

	log.Println("Scanning for Bose devices...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	svc := discovery.NewService(10 * time.Second)
	devices, err := svc.DiscoverDevices(ctx)
	if err != nil {
		log.Printf("Discovery error: %v", err)
		return
	}

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
		if err := s.ds.SaveDeviceInfo("default", d.SerialNo, info); err != nil {
			log.Printf("Failed to save device info: %v", err)
		}
	}
}

func (s *Server) handleProxyRequest(w http.ResponseWriter, r *http.Request) {
	targetURLStr := strings.TrimPrefix(r.URL.Path, "/proxy/")
	if targetURLStr == "" {
		http.Error(w, "Target URL is required", http.StatusBadRequest)
		return
	}

	// Reconstruct original URL (it might have lost its double slashes in the path)
	if !strings.HasPrefix(targetURLStr, "http://") && !strings.HasPrefix(targetURLStr, "https://") {
		// Try to fix it if it looks like http:/...
		if strings.HasPrefix(targetURLStr, "http:/") {
			targetURLStr = "http://" + strings.TrimPrefix(targetURLStr, "http:/")
		} else if strings.HasPrefix(targetURLStr, "https:/") {
			targetURLStr = "https://" + strings.TrimPrefix(targetURLStr, "https:/")
		}
	}

	target, err := url.Parse(targetURLStr)
	if err != nil {
		http.Error(w, "Invalid target URL: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("Proxying request: %s %s -> %s", r.Method, r.URL.Path, target.String())

	proxy := httputil.NewSingleHostReverseProxy(target)
	// Update director to set the correct host and path
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.URL.Path = target.Path
		req.URL.RawQuery = r.URL.RawQuery
	}

	proxy.ServeHTTP(w, r)
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

	serverURL := os.Getenv("SERVER_URL")
	if serverURL == "" {
		// Try to guess the server URL
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "localhost"
		}
		serverURL = "http://" + hostname + ":" + port
	}

	sm := setup.NewManager(serverURL, ds)

	server := &Server{
		ds:        ds,
		sm:        sm,
		serverURL: serverURL,
	}

	proxyPort := os.Getenv("PROXY_PORT")
	if proxyPort == "" {
		proxyPort = "8080"
	}
	proxyAddr := bindAddr + ":" + proxyPort
	if bindAddr == "" {
		proxyAddr = ":" + proxyPort
	}

	// Guess proxyURL
	server.proxyURL = os.Getenv("PROXY_URL")
	if server.proxyURL == "" {
		u, _ := url.Parse(serverURL)
		host, _, _ := net.SplitHostPort(u.Host)
		if host == "" {
			host = u.Host
		}
		server.proxyURL = "http://" + host + ":" + proxyPort
	}

	// Start Proxy Server
	go func() {
		proxyRouter := chi.NewRouter()
		proxyRouter.Use(middleware.Logger)
		proxyRouter.Get("/proxy/*", server.handleProxyRequest)
		log.Printf("Proxy service starting on %s", proxyAddr)
		if err := http.ListenAndServe(proxyAddr, proxyRouter); err != nil {
			log.Printf("Proxy server error: %v", err)
		}
	}()

	// Phase 5: Device Discovery
	go func() {
		for {
			server.discoverDevices()
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

	// Phase 7: Setup and Discovery endpoints
	r.Route("/setup", func(r chi.Router) {
		r.Get("/devices", server.handleListDiscoveredDevices)
		r.Post("/discover", server.handleTriggerDiscovery)
		r.Get("/discovery-status", server.handleGetDiscoveryStatus)
		r.Get("/settings", server.handleGetSettings)
		r.Get("/info/{deviceIP}", server.handleGetDeviceInfo)
		r.Get("/summary/{deviceIP}", server.handleGetMigrationSummary)
		r.Post("/migrate/{deviceIP}", server.handleMigrateDevice)
		r.Post("/ensure-remote-services/{deviceIP}", server.handleEnsureRemoteServices)
		r.Post("/backup/{deviceIP}", server.handleBackupConfig)
	})

	// Delegation Logic: Proxy everything else to Python
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Proxying request: %s %s -> %s", r.Method, r.URL.Path, targetURL)
		proxy.ServeHTTP(w, r)
	})

	log.Printf("Go service starting on %s, proxying to %s", addr, targetURL)
	log.Fatal(http.ListenAndServe(addr, r))
}
