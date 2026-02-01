package main

import (
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/deborahgu/soundcork/internal/marge"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleMargeSourceProviders(w http.ResponseWriter, r *http.Request) {
	data, err := marge.SourceProvidersToXML()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Write(data)
}

func (s *Server) handleMargeAccountFull(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "account")
	data, err := marge.AccountFullToXML(s.ds, account)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Write(data)
}

func (s *Server) handleMargePowerOn(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleMargeSoftwareUpdate(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("../soundcork/swupdate.xml")
	if err != nil {
		data, err = os.ReadFile("soundcork/swupdate.xml")
	}
	if err != nil {
		w.Header().Set("Content-Type", "application/xml")
		w.Write([]byte(marge.SoftwareUpdateToXML()))
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Write(data)
}

func (s *Server) handleMargePresets(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "account")
	data, err := marge.PresetsToXML(s.ds, account)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Write(data)
}

func (s *Server) handleMargeUpdatePreset(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "account")
	device := chi.URLParam(r, "device")
	presetNumberStr := chi.URLParam(r, "presetNumber")
	presetNumber, err := strconv.Atoi(presetNumberStr)
	if err != nil {
		http.Error(w, "Invalid preset number", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	data, err := marge.UpdatePreset(s.ds, account, device, presetNumber, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Write(data)
}

func (s *Server) handleMargeAddRecent(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "account")
	device := chi.URLParam(r, "device")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	data, err := marge.AddRecent(s.ds, account, device, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Write(data)
}

func (s *Server) handleMargeAddDevice(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "account")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		return
	}
	data, err := marge.AddDeviceToAccount(s.ds, account, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Write(data)
}

func (s *Server) handleMargeRemoveDevice(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "account")
	device := chi.URLParam(r, "device")
	if err := marge.RemoveDeviceFromAccount(s.ds, account, device); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok": true}`))
}
