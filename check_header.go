package main

import (
	"fmt"
	"net/http"
)

func main() {
	etag := "12345"
	fmt.Printf("Normalized ETag: %s\n", http.CanonicalHeaderKey("ETag"))
	fmt.Printf("Normalized Etag: %s\n", http.CanonicalHeaderKey("Etag"))
	fmt.Printf("Normalized E-Tag: %s\n", http.CanonicalHeaderKey("E-Tag"))

	h := make(http.Header)
	h["ETag"] = []string{etag}
	fmt.Printf("Map with ETag: %v\n", h)

	h.Set("ETag", etag)
	fmt.Printf("Map after Set(ETag): %v\n", h)
}
