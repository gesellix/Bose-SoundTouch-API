package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func setupRouter(targetURL string) *chi.Mux {
	target, _ := url.Parse(targetURL)
	proxy := &reverseProxy{target: target}

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"Bose": "Can't Brick Us", "service": "Go/Chi"}`))
	})

	// Setup media directory for tests
	mediaDir := http.Dir("../../soundcork/media")
	r.Get("/media/*", func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/media/", http.FileServer(mediaDir))
		fs.ServeHTTP(w, r)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})
	return r
}

type reverseProxy struct {
	target *url.URL
}

func (p *reverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Simplified proxy for testing
	w.WriteHeader(http.StatusAccepted) // Custom status to identify proxy hit in tests
	w.Write([]byte("Proxied to " + p.target.String()))
}

func TestRootEndpoint(t *testing.T) {
	r := setupRouter("http://localhost:8001")
	ts := httptest.NewServer(r)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", res.Status)
	}

	body, _ := io.ReadAll(res.Body)
	expected := `{"Bose": "Can't Brick Us", "service": "Go/Chi"}`
	if string(body) != expected {
		t.Errorf("Expected body %s, got %s", expected, string(body))
	}
}

func TestProxyDelegation(t *testing.T) {
	r := setupRouter("http://localhost:8001")
	ts := httptest.NewServer(r)
	defer ts.Close()

	res, err := http.Get(ts.URL + "/some-other-endpoint")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	// In our mock, proxy returns StatusAccepted (202)
	if res.StatusCode != http.StatusAccepted {
		t.Errorf("Expected status %v (Proxy Hit), got %v", http.StatusAccepted, res.StatusCode)
	}

	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "Proxied to http://localhost:8001") {
		t.Errorf("Expected proxy message, got %s", string(body))
	}
}

func TestStaticMedia(t *testing.T) {
	r := setupRouter("http://localhost:8001")
	ts := httptest.NewServer(r)
	defer ts.Close()

	// Use a known file from soundcork/media
	res, err := http.Get(ts.URL + "/media/SiriusXM_Logo_Color.svg")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", res.Status)
	}

	contentType := res.Header.Get("Content-Type")
	if !strings.Contains(contentType, "image/svg+xml") {
		t.Errorf("Expected image/svg+xml content type, got %s", contentType)
	}
}
