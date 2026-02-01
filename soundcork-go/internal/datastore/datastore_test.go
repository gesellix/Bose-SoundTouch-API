package datastore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/deborahgu/soundcork/internal/models"
)

func TestDataStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "soundcork-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	ds := NewDataStore(tempDir)
	account := "test-account"
	device := "test-device"

	// Test Save/Get DeviceInfo
	info := &models.DeviceInfo{
		DeviceID: device,
		Name:     "Test Speaker",
	}
	err = ds.SaveDeviceInfo(account, device, info)
	if err != nil {
		t.Errorf("SaveDeviceInfo failed: %v", err)
	}

	loadedInfo, err := ds.GetDeviceInfo(account, device)
	if err != nil {
		t.Errorf("GetDeviceInfo failed: %v", err)
	}
	if loadedInfo.Name != info.Name {
		t.Errorf("Expected name %s, got %s", info.Name, loadedInfo.Name)
	}

	// Test Presets
	presets := []models.Preset{
		{
			ContentItem: models.ContentItem{
				Name: "Preset 1",
			},
		},
	}
	err = ds.SavePresets(account, presets)
	if err != nil {
		t.Errorf("SavePresets failed: %v", err)
	}

	loadedPresets, err := ds.GetPresets(account)
	if err != nil {
		t.Errorf("GetPresets failed: %v", err)
	}
	if len(loadedPresets) != 1 || loadedPresets[0].ContentItem.Name != "Preset 1" {
		t.Errorf("Unexpected presets: %+v", loadedPresets)
	}

	// Test Recents
	recents := []models.Recent{
		{
			ContentItem: models.ContentItem{
				Name: "Recent 1",
			},
		},
	}
	err = ds.SaveRecents(account, recents)
	if err != nil {
		t.Errorf("SaveRecents failed: %v", err)
	}

	loadedRecents, err := ds.GetRecents(account)
	if err != nil {
		t.Errorf("GetRecents failed: %v", err)
	}
	if len(loadedRecents) != 1 || loadedRecents[0].ContentItem.Name != "Recent 1" {
		t.Errorf("Unexpected recents: %+v", loadedRecents)
	}

	// Test path helpers
	expectedAccountDir := filepath.Join(tempDir, account)
	if ds.AccountDir(account) != expectedAccountDir {
		t.Errorf("Expected account dir %s, got %s", expectedAccountDir, ds.AccountDir(account))
	}
}
