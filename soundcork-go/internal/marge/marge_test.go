package marge

import (
	"strings"
	"testing"

	"os"

	"github.com/deborahgu/soundcork/internal/datastore"
	"github.com/deborahgu/soundcork/internal/models"
)

func TestMargeXML(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "marge-test-*")
	defer os.RemoveAll(tempDir)
	ds := datastore.NewDataStore(tempDir)
	account := "123"
	device := "ABC"

	// Setup initial data
	info := &models.DeviceInfo{
		DeviceID: device,
		Name:     "Living Room",
	}
	ds.SaveDeviceInfo(account, device, info)

	// Save empty presets/recents to avoid index out of range when stripping header
	ds.SavePresets(account, []models.Preset{})
	ds.SaveRecents(account, []models.Recent{})

	// Test SourceProvidersToXML
	xmlData, err := SourceProvidersToXML()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(xmlData), "<sourceProviders>") {
		t.Errorf("Expected <sourceProviders>, got %s", string(xmlData))
	}

	// Test AccountFullToXML
	fullXML, err := AccountFullToXML(ds, account)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fullXML), `id="123"`) {
		t.Errorf("Expected account id 123, got %s", string(fullXML))
	}
	if !strings.Contains(string(fullXML), "Living Room") {
		t.Errorf("Expected device name Living Room, got %s", string(fullXML))
	}

	// Test SoftwareUpdateToXML
	swXML := SoftwareUpdateToXML()
	if !strings.Contains(swXML, "<software_update>") {
		t.Errorf("Expected <software_update>, got %s", swXML)
	}
}
