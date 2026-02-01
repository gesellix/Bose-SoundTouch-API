package bmx

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestPlayCustomStream(t *testing.T) {
	// Simple test for custom stream XML generation
	dataObj := struct {
		StreamURL string `json:"streamUrl"`
		ImageURL  string `json:"imageUrl"`
		Name      string `json:"name"`
	}{
		StreamURL: "http://example.com/stream.mp3",
		ImageURL:  "image.png",
		Name:      "Stream Name",
	}
	jsonBytes, _ := json.Marshal(dataObj)
	data := base64.StdEncoding.EncodeToString(jsonBytes)

	resp, err := PlayCustomStream(data)
	if err != nil {
		t.Fatalf("PlayCustomStream failed: %v", err)
	}

	if resp.Audio.StreamUrl != "http://example.com/stream.mp3" {
		t.Errorf("Expected stream URL http://example.com/stream.mp3, got %s", resp.Audio.StreamUrl)
	}
	if resp.Name != "Stream Name" {
		t.Errorf("Expected name Stream Name, got %s", resp.Name)
	}
	if resp.ImageUrl != "image.png" {
		t.Errorf("Expected image URL image.png, got %s", resp.ImageUrl)
	}
}
