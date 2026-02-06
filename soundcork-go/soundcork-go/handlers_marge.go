package main

import (
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/deborahgu/soundcork/internal/marge"
	"github.com/go-chi/chi/v5"
)

func (s *Server) handleMargeSourceProviders(w http.ResponseWriter, r *http.Request) {
	etag := strconv.FormatInt(time.Now().UnixMilli(), 10)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	data, err := marge.SourceProvidersToXML()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("ETag", etag)
	w.Write(data)
}

func (s *Server) handleMargeAccountFull(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "account")
	etag := strconv.FormatInt(s.ds.GetETagForAccount(account), 10)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	data, err := marge.AccountFullToXML(s.ds, account)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("ETag", etag)
	w.Write(data)
}

func (s *Server) handleMargePowerOn(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleMargeSoftwareUpdate(w http.ResponseWriter, r *http.Request) {
	path := "../soundcork/swupdate.xml"
	if _, err := os.Stat(path); err != nil {
		path = "soundcork/swupdate.xml"
	}

	var etag string
	if info, err := os.Stat(path); err == nil {
		etag = strconv.FormatInt(info.ModTime().UnixMilli(), 10)
	} else {
		etag = "default"
	}

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("ETag", etag)
		w.Write([]byte(marge.SoftwareUpdateToXML()))
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("ETag", etag)
	w.Write(data)
}

func (s *Server) handleMargePresets(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "account")
	etag := strconv.FormatInt(s.ds.GetETagForPresets(account), 10)
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	data, err := marge.PresetsToXML(s.ds, account)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("ETag", etag)
	w.Write(data)
}

func (s *Server) handleMargeUpdatePreset(w http.ResponseWriter, r *http.Request) {
	account := chi.URLParam(r, "account")
	device := chi.URLParam(r, "device")

	etag := strconv.FormatInt(s.ds.GetETagForPresets(account), 10)
	w.Header().Set("ETag", etag)

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

	etag := strconv.FormatInt(s.ds.GetETagForRecents(account), 10)
	w.Header().Set("ETag", etag)

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
