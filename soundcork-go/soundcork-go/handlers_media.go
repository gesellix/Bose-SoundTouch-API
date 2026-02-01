package main

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"
)

//go:embed index.html
var indexHTML []byte

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	accept := r.Header.Get("Accept")
	if !strings.Contains(accept, "text/html") && (strings.Contains(accept, "application/json") || accept == "*/*" || accept == "") {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"Bose": "Can't Brick Us", "service": "Go/Chi"}`)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(indexHTML)
}

func (s *Server) handleMedia(mediaDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fs := http.StripPrefix("/media/", http.FileServer(http.Dir(mediaDir)))
		fs.ServeHTTP(w, r)
	}
}
