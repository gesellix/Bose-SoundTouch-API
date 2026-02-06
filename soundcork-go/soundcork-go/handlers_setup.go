package main

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListDiscoveredDevices(w http.ResponseWriter, r *http.Request) {
	devices, err := s.ds.ListAllDevices()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) handleTriggerDiscovery(w http.ResponseWriter, r *http.Request) {
	go s.discoverDevices()
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "Discovery started"}`))
}

func (s *Server) handleGetDiscoveryStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"discovering": s.discovering})
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"server_url": s.serverURL,
	})
}

func (s *Server) handleGetDeviceInfo(w http.ResponseWriter, r *http.Request) {
	deviceIP := chi.URLParam(r, "deviceIP")
	if deviceIP == "" {
		http.Error(w, "Device IP is required", http.StatusBadRequest)
		return
	}

	info, err := s.sm.GetLiveDeviceInfo(deviceIP)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (s *Server) handleGetMigrationSummary(w http.ResponseWriter, r *http.Request) {
	deviceIP := chi.URLParam(r, "deviceIP")
	if deviceIP == "" {
		http.Error(w, "Device IP is required", http.StatusBadRequest)
		return
	}

	targetURL := r.URL.Query().Get("target_url")

	summary, err := s.sm.GetMigrationSummary(deviceIP, targetURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (s *Server) handleMigrateDevice(w http.ResponseWriter, r *http.Request) {
	deviceIP := chi.URLParam(r, "deviceIP")
	if deviceIP == "" {
		http.Error(w, "Device IP is required", http.StatusBadRequest)
		return
	}

	targetURL := r.URL.Query().Get("target_url")

	if err := s.sm.MigrateSpeaker(deviceIP, targetURL); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok": true, "message": "Migration started"}`))
}

func (s *Server) handleEnsureRemoteServices(w http.ResponseWriter, r *http.Request) {
	deviceIP := chi.URLParam(r, "deviceIP")
	if deviceIP == "" {
		http.Error(w, "Device IP is required", http.StatusBadRequest)
		return
	}

	if err := s.sm.EnsureRemoteServices(deviceIP); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok": true, "message": "Remote services ensured"}`))
}
