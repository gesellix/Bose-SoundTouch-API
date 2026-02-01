package datastore

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/deborahgu/soundcork/internal/constants"
	"github.com/deborahgu/soundcork/internal/models"
)

type DataStore struct {
	DataDir string
}

func NewDataStore(dataDir string) *DataStore {
	if dataDir == "" {
		dataDir = "data"
	}
	return &DataStore{DataDir: dataDir}
}

func (ds *DataStore) AccountDir(account string) string {
	return filepath.Join(ds.DataDir, account)
}

func (ds *DataStore) AccountDevicesDir(account string) string {
	return filepath.Join(ds.DataDir, account, constants.DevicesDir)
}

func (ds *DataStore) AccountDeviceDir(account, device string) string {
	return filepath.Join(ds.AccountDevicesDir(account), device)
}

func (ds *DataStore) GetDeviceInfo(account, device string) (*models.DeviceInfo, error) {
	path := filepath.Join(ds.AccountDeviceDir(account, device), constants.DeviceInfoFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var info struct {
		DeviceID   string `xml:"deviceID,attr"`
		Name       string `xml:"name"`
		Type       string `xml:"type"`
		ModuleType string `xml:"moduleType"`
		Components []struct {
			Category        string `xml:"componentCategory"`
			SoftwareVersion string `xml:"softwareVersion"`
			SerialNumber    string `xml:"serialNumber"`
		} `xml:"components>component"`
		NetworkInfo []struct {
			Type      string `xml:"type,attr"`
			IPAddress string `xml:"ipAddress"`
		} `xml:"networkInfo"`
	}

	if err := xml.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	deviceInfo := &models.DeviceInfo{
		DeviceID:    info.DeviceID,
		ProductCode: fmt.Sprintf("%s %s", info.Type, info.ModuleType),
		Name:        info.Name,
	}

	for _, comp := range info.Components {
		if comp.Category == "SCM" {
			deviceInfo.FirmwareVersion = comp.SoftwareVersion
			deviceInfo.DeviceSerialNumber = comp.SerialNumber
		} else if comp.Category == "PackagedProduct" {
			deviceInfo.ProductSerialNumber = comp.SerialNumber
		}
	}

	for _, net := range info.NetworkInfo {
		if net.Type == "SCM" {
			deviceInfo.IPAddress = net.IPAddress
		}
	}

	return deviceInfo, nil
}

func (ds *DataStore) GetPresets(account string) ([]models.Preset, error) {
	path := filepath.Join(ds.AccountDir(account), constants.PresetsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var presetsWrap struct {
		Presets []struct {
			ID          string `xml:"id,attr"`
			CreatedOn   string `xml:"createdOn,attr"`
			UpdatedOn   string `xml:"updatedOn,attr"`
			ContentItem struct {
				Source        string `xml:"source,attr"`
				Type          string `xml:"type,attr"`
				Location      string `xml:"location,attr"`
				SourceAccount string `xml:"sourceAccount,attr"`
				IsPresetable  string `xml:"isPresetable,attr"`
				ItemName      string `xml:"itemName"`
				ContainerArt  string `xml:"containerArt"`
			} `xml:"ContentItem"`
		} `xml:"preset"`
	}

	if err := xml.Unmarshal(data, &presetsWrap); err != nil {
		return nil, err
	}

	var presets []models.Preset
	for _, p := range presetsWrap.Presets {
		presets = append(presets, models.Preset{
			ContentItem: models.ContentItem{
				ID:            p.ID,
				Name:          p.ContentItem.ItemName,
				Source:        p.ContentItem.Source,
				Type:          p.ContentItem.Type,
				Location:      p.ContentItem.Location,
				SourceAccount: p.ContentItem.SourceAccount,
				IsPresetable:  p.ContentItem.IsPresetable,
			},
			ContainerArt: p.ContentItem.ContainerArt,
			CreatedOn:    p.CreatedOn,
			UpdatedOn:    p.UpdatedOn,
		})
	}

	return presets, nil
}

func (ds *DataStore) SavePresets(account string, presets []models.Preset) error {
	path := filepath.Join(ds.AccountDir(account), constants.PresetsFile)

	type PresetXML struct {
		ID          string `xml:"id,attr"`
		CreatedOn   string `xml:"createdOn,attr"`
		UpdatedOn   string `xml:"updatedOn,attr"`
		ContentItem struct {
			Source        string `xml:"source,attr,omitempty"`
			Type          string `xml:"type,attr"`
			Location      string `xml:"location,attr"`
			SourceAccount string `xml:"sourceAccount,attr,omitempty"`
			IsPresetable  string `xml:"isPresetable,attr"`
			ItemName      string `xml:"itemName"`
			ContainerArt  string `xml:"containerArt"`
		} `xml:"ContentItem"`
	}

	type PresetsXML struct {
		XMLName xml.Name    `xml:"presets"`
		Presets []PresetXML `xml:"preset"`
	}

	var px PresetsXML
	for _, p := range presets {
		var pxml PresetXML
		pxml.ID = p.ID
		pxml.CreatedOn = p.CreatedOn
		pxml.UpdatedOn = p.UpdatedOn
		pxml.ContentItem.Source = p.Source
		pxml.ContentItem.Type = p.Type
		pxml.ContentItem.Location = p.Location
		pxml.ContentItem.SourceAccount = p.SourceAccount
		pxml.ContentItem.IsPresetable = "true"
		pxml.ContentItem.ItemName = p.Name
		pxml.ContentItem.ContainerArt = p.ContainerArt
		px.Presets = append(px.Presets, pxml)
	}

	data, err := xml.MarshalIndent(px, "", "    ")
	if err != nil {
		return err
	}

	header := []byte(xml.Header)
	return os.WriteFile(path, append(header, data...), 0644)
}

func (ds *DataStore) GetRecents(account string) ([]models.Recent, error) {
	path := filepath.Join(ds.AccountDir(account), constants.RecentsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var recentsWrap struct {
		Recents []struct {
			ID          string `xml:"id,attr"`
			DeviceID    string `xml:"deviceID,attr"`
			UtcTime     string `xml:"utcTime,attr"`
			ContentItem struct {
				Source        string `xml:"source,attr"`
				Type          string `xml:"type,attr"`
				Location      string `xml:"location,attr"`
				SourceAccount string `xml:"sourceAccount,attr"`
				IsPresetable  string `xml:"isPresetable,attr"`
				ItemName      string `xml:"itemName"`
				ContainerArt  string `xml:"containerArt"`
			} `xml:"contentItem"`
		} `xml:"recent"`
	}

	if err := xml.Unmarshal(data, &recentsWrap); err != nil {
		return nil, err
	}

	var recents []models.Recent
	for _, r := range recentsWrap.Recents {
		recents = append(recents, models.Recent{
			ContentItem: models.ContentItem{
				ID:            r.ID,
				Name:          r.ContentItem.ItemName,
				Source:        r.ContentItem.Source,
				Type:          r.ContentItem.Type,
				Location:      r.ContentItem.Location,
				SourceAccount: r.ContentItem.SourceAccount,
				IsPresetable:  r.ContentItem.IsPresetable,
			},
			DeviceID:     r.DeviceID,
			UtcTime:      r.UtcTime,
			ContainerArt: r.ContentItem.ContainerArt,
		})
	}

	return recents, nil
}

func (ds *DataStore) SaveRecents(account string, recents []models.Recent) error {
	path := filepath.Join(ds.AccountDir(account), constants.RecentsFile)

	type RecentXML struct {
		ID          string `xml:"id,attr"`
		DeviceID    string `xml:"deviceID,attr"`
		UtcTime     string `xml:"utcTime,attr"`
		ContentItem struct {
			Source        string `xml:"source,attr,omitempty"`
			Type          string `xml:"type,attr"`
			Location      string `xml:"location,attr"`
			SourceAccount string `xml:"sourceAccount,attr,omitempty"`
			IsPresetable  string `xml:"isPresetable,attr"`
			ItemName      string `xml:"itemName"`
			ContainerArt  string `xml:"containerArt"`
		} `xml:"contentItem"`
	}

	type RecentsXML struct {
		XMLName xml.Name    `xml:"recents"`
		Recents []RecentXML `xml:"recent"`
	}

	var rx RecentsXML
	for _, r := range recents {
		var rxml RecentXML
		rxml.ID = r.ID
		rxml.DeviceID = r.DeviceID
		rxml.UtcTime = r.UtcTime
		rxml.ContentItem.Source = r.Source
		rxml.ContentItem.Type = r.Type
		rxml.ContentItem.Location = r.Location
		rxml.ContentItem.SourceAccount = r.SourceAccount
		rxml.ContentItem.IsPresetable = r.IsPresetable
		if rxml.ContentItem.IsPresetable == "" {
			rxml.ContentItem.IsPresetable = "true"
		}
		rxml.ContentItem.ItemName = r.Name
		rxml.ContentItem.ContainerArt = r.ContainerArt
		rx.Recents = append(rx.Recents, rxml)
	}

	data, err := xml.MarshalIndent(rx, "", "    ")
	if err != nil {
		return err
	}

	header := []byte(xml.Header)
	return os.WriteFile(path, append(header, data...), 0644)
}

func (ds *DataStore) SaveDeviceInfo(account string, device string, info *models.DeviceInfo) error {
	dir := ds.AccountDeviceDir(account, device)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	path := filepath.Join(dir, constants.DeviceInfoFile)

	type ComponentXML struct {
		ComponentCategory string `xml:"componentCategory"`
		SoftwareVersion   string `xml:"softwareVersion,omitempty"`
		SerialNumber      string `xml:"serialNumber,omitempty"`
	}

	type NetworkInfoXML struct {
		Type      string `xml:"type,attr"`
		IPAddress string `xml:"ipAddress"`
	}

	type InfoXML struct {
		XMLName     xml.Name         `xml:"info"`
		DeviceID    string           `xml:"deviceID,attr"`
		Name        string           `xml:"name"`
		Type        string           `xml:"type"`
		ModuleType  string           `xml:"moduleType"`
		Components  []ComponentXML   `xml:"components>component"`
		NetworkInfo []NetworkInfoXML `xml:"networkInfo"`
	}

	// Parsing product code back to type and moduleType (best effort)
	// Python: f"{type} {module_type}"
	devType := info.ProductCode
	moduleType := ""
	for i := 0; i < len(info.ProductCode); i++ {
		if info.ProductCode[i] == ' ' {
			devType = info.ProductCode[:i]
			moduleType = info.ProductCode[i+1:]
			break
		}
	}

	ix := InfoXML{
		DeviceID:   info.DeviceID,
		Name:       info.Name,
		Type:       devType,
		ModuleType: moduleType,
		Components: []ComponentXML{
			{
				ComponentCategory: "SCM",
				SoftwareVersion:   info.FirmwareVersion,
				SerialNumber:      info.DeviceSerialNumber,
			},
			{
				ComponentCategory: "PackagedProduct",
				SerialNumber:      info.ProductSerialNumber,
			},
		},
		NetworkInfo: []NetworkInfoXML{
			{
				Type:      "SCM",
				IPAddress: info.IPAddress,
			},
		},
	}

	data, err := xml.MarshalIndent(ix, "", "    ")
	if err != nil {
		return err
	}

	header := []byte(xml.Header)
	return os.WriteFile(path, append(header, data...), 0644)
}

func (ds *DataStore) RemoveDevice(account string, device string) error {
	dir := ds.AccountDeviceDir(account, device)
	return os.RemoveAll(dir)
}

func (ds *DataStore) GetConfiguredSources(account string) ([]models.ConfiguredSource, error) {
	path := filepath.Join(ds.AccountDir(account), constants.SourcesFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var sourcesWrap struct {
		Sources []struct {
			DisplayName string `xml:"displayName,attr"`
			ID          string `xml:"id,attr"`
			Secret      string `xml:"secret,attr"`
			SecretType  string `xml:"secretType,attr"`
			SourceKey   struct {
				Account string `xml:"account,attr"`
				Type    string `xml:"type,attr"`
			} `xml:"sourceKey"`
		} `xml:"source"`
	}

	if err := xml.Unmarshal(data, &sourcesWrap); err != nil {
		return nil, err
	}

	var sources []models.ConfiguredSource
	lastID := 100001
	for _, s := range sourcesWrap.Sources {
		id := s.ID
		if id == "" {
			id = strconv.Itoa(lastID)
			lastID++
		}
		sources = append(sources, models.ConfiguredSource{
			DisplayName:      s.DisplayName,
			ID:               id,
			Secret:           s.Secret,
			SecretType:       s.SecretType,
			SourceKeyType:    s.SourceKey.Type,
			SourceKeyAccount: s.SourceKey.Account,
		})
	}

	return sources, nil
}

func (ds *DataStore) GetETagForPresets(account string) int64 {
	path := filepath.Join(ds.AccountDir(account), constants.PresetsFile)
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixNano() / int64(time.Millisecond)
}

func (ds *DataStore) GetETagForSources(account string) int64 {
	path := filepath.Join(ds.AccountDir(account), constants.SourcesFile)
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixNano() / int64(time.Millisecond)
}

func (ds *DataStore) GetETagForRecents(account string) int64 {
	path := filepath.Join(ds.AccountDir(account), constants.RecentsFile)
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.ModTime().UnixNano() / int64(time.Millisecond)
}

func (ds *DataStore) GetETagForAccount(account string) int64 {
	e1 := ds.GetETagForPresets(account)
	e2 := ds.GetETagForSources(account)
	e3 := ds.GetETagForRecents(account)
	max := e1
	if e2 > max {
		max = e2
	}
	if e3 > max {
		max = e3
	}
	return max
}
