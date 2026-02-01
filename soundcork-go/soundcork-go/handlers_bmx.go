package main

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/deborahgu/soundcork/internal/bmx"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleBMXRegistry(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("soundcork/bmx_services.json")
	if err != nil {
		data, err = os.ReadFile("../soundcork/bmx_services.json")
	}
	if err != nil {
		data, err = os.ReadFile("../../soundcork/bmx_services.json")
	}
	if err != nil {
		http.Error(w, "Failed to read services", http.StatusInternalServerError)
		return
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}

	content := string(data)
	content = strings.ReplaceAll(content, "{BMX_SERVER}", baseURL)
	content = strings.ReplaceAll(content, "{MEDIA_SERVER}", baseURL+"/media")

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(content))
}

func (s *Server) handleTuneInPlayback(w http.ResponseWriter, r *http.Request) {
	stationID := chi.URLParam(r, "stationID")
	resp, err := bmx.TuneInPlayback(stationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleTuneInPodcastInfo(w http.ResponseWriter, r *http.Request) {
	podcastID := chi.URLParam(r, "podcastID")
	encodedName := r.URL.Query().Get("encoded_name")
	resp, err := bmx.TuneInPodcastInfo(podcastID, encodedName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleTuneInPlaybackPodcast(w http.ResponseWriter, r *http.Request) {
	podcastID := chi.URLParam(r, "podcastID")
	resp, err := bmx.TuneInPlaybackPodcast(podcastID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleOrionPlayback(w http.ResponseWriter, r *http.Request) {
	data := chi.URLParam(r, "data")
	resp, err := bmx.PlayCustomStream(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
