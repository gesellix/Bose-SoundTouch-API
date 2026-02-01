package main

import (
	"fmt"
	"net/http"
)

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"Bose": "Can't Brick Us", "service": "Go/Chi"}`)
}

func (s *Server) handleMedia(mediaDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/media/", http.FileServer(http.Dir(mediaDir)))
		fs.ServeHTTP(w, r)
	}
}
