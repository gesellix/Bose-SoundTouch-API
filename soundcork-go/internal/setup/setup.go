package setup

import (
	"encoding/xml"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/deborahgu/soundcork/internal/datastore"
	"github.com/deborahgu/soundcork/internal/ssh"
)

const SoundTouchSdkPrivateCfgPath = "/opt/Bose/etc/SoundTouchSdkPrivateCfg.xml"

// PrivateCfg represents the SoundTouchSdkPrivateCfg XML structure.
type PrivateCfg struct {
	XMLName                    xml.Name `xml:"SoundTouchSdkPrivateCfg"`
	MargeServerUrl             string   `xml:"margeServerUrl"`
	StatsServerUrl             string   `xml:"statsServerUrl"`
	SwUpdateUrl                string   `xml:"swUpdateUrl"`
	UsePandoraProductionServer bool     `xml:"usePandoraProductionServer"`
	IsZeroconfEnabled          bool     `xml:"isZeroconfEnabled"`
	SaveMargeCustomerReport    bool     `xml:"saveMargeCustomerReport"`
	BmxRegistryUrl             string   `xml:"bmxRegistryUrl"`
}

// MigrationSummary provides details about the state of a speaker before migration.
type MigrationSummary struct {
	SSHSuccess               bool     `json:"ssh_success"`
	CurrentConfig            string   `json:"current_config"`
	PlannedConfig            string   `json:"planned_config"`
	RemoteServicesEnabled    bool     `json:"remote_services_enabled"`
	RemoteServicesPersistent bool     `json:"remote_services_persistent"`
	RemoteServicesFound      []string `json:"remote_services_found"`
	RemoteServicesCheckErr   string   `json:"remote_services_check_err,omitempty"`
	DeviceName               string   `json:"device_name,omitempty"`
	DeviceModel              string   `json:"device_model,omitempty"`
	DeviceSerial             string   `json:"device_serial,omitempty"`
	FirmwareVersion          string   `json:"firmware_version,omitempty"`
}

// Manager handles the migration of speakers to the soundcork service.
type Manager struct {
	ServerURL string
	DataStore *datastore.DataStore
}

// NewManager creates a new Manager with the given base server URL.
func NewManager(serverURL string, ds *datastore.DataStore) *Manager {
	return &Manager{ServerURL: serverURL, DataStore: ds}
}

// DeviceInfoXML represents the XML structure from :8090/info
type DeviceInfoXML struct {
	XMLName      xml.Name `xml:"info" json:"-"`
	DeviceID     string   `xml:"deviceID,attr" json:"deviceID"`
	Name         string   `xml:"name" json:"name"`
	Type         string   `xml:"type" json:"type"`
	MaccAddress  string   `xml:"maccAddress" json:"maccAddress"`
	SoftwareVer  string   `xml:"-" json:"softwareVersion"`
	SerialNumber string   `xml:"-" json:"serialNumber"`
	Components   []struct {
		Category        string `xml:"componentCategory"`
		SoftwareVersion string `xml:"softwareVersion"`
		SerialNumber    string `xml:"serialNumber"`
	} `xml:"components>component" json:"-"`
}

// GetLiveDeviceInfo fetches live information from the speaker's :8090/info endpoint.
func (m *Manager) GetLiveDeviceInfo(deviceIP string) (*DeviceInfoXML, error) {
	infoURL := fmt.Sprintf("http://%s:8090/info", deviceIP)
	// For testing, if the IP already contains a port, don't append :8090
	if host, _, err := net.SplitHostPort(deviceIP); err == nil {
		infoURL = fmt.Sprintf("http://%s/info", deviceIP)
		_ = host
	}
	resp, err := http.Get(infoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch info from %s: %v", infoURL, err)
	}
	defer resp.Body.Close()

	var infoXML DeviceInfoXML
	if err := xml.NewDecoder(resp.Body).Decode(&infoXML); err != nil {
		return nil, fmt.Errorf("failed to decode info XML from %s: %v", infoURL, err)
	}

	for _, comp := range infoXML.Components {
		if comp.Category == "SCM" {
			infoXML.SoftwareVer = comp.SoftwareVersion
			if infoXML.SerialNumber == "" {
				infoXML.SerialNumber = comp.SerialNumber
			}
		} else if comp.Category == "PackagedProduct" {
			if infoXML.SerialNumber == "" {
				infoXML.SerialNumber = comp.SerialNumber
			}
		}
	}

	return &infoXML, nil
}

// GetMigrationSummary returns a summary of the current and planned state of the speaker.
func (m *Manager) GetMigrationSummary(deviceIP string, targetURL string) (*MigrationSummary, error) {
	if targetURL == "" {
		targetURL = m.ServerURL
	}
	client := ssh.NewClient(deviceIP)

	summary := &MigrationSummary{
		SSHSuccess: false,
	}

	// 0. Populate from datastore if available
	if m.DataStore != nil {
		devices, err := m.DataStore.ListAllDevices()
		if err == nil {
			for _, d := range devices {
				if d.IPAddress == deviceIP {
					summary.DeviceName = d.Name
					summary.DeviceModel = d.ProductCode
					summary.DeviceSerial = d.DeviceSerialNumber
					summary.FirmwareVersion = d.FirmwareVersion
					break
				}
			}
		} else {
			log.Printf("Warning: failed to list devices from datastore: %v", err)
		}
	}

	// 0a. Supplement with live info from :8090/info
	infoXML, err := m.GetLiveDeviceInfo(deviceIP)
	if err == nil {
		if infoXML.Name != "" {
			summary.DeviceName = infoXML.Name
		}
		if infoXML.Type != "" {
			summary.DeviceModel = infoXML.Type
		}
		if infoXML.SerialNumber != "" {
			summary.DeviceSerial = infoXML.SerialNumber
		}
		if infoXML.SoftwareVer != "" {
			summary.FirmwareVersion = infoXML.SoftwareVer
		}
	} else {
		log.Printf("Warning: %v", err)
	}

	// 1. Generate planned config (we can do this anyway)
	plannedCfg := PrivateCfg{
		MargeServerUrl:             fmt.Sprintf("%s/marge", targetURL),
		StatsServerUrl:             targetURL,
		SwUpdateUrl:                fmt.Sprintf("%s/updates/soundtouch", targetURL),
		UsePandoraProductionServer: true,
		IsZeroconfEnabled:          true,
		SaveMargeCustomerReport:    false,
		BmxRegistryUrl:             fmt.Sprintf("%s/bmx/registry/v1/services", targetURL),
	}

	xmlContent, err := xml.MarshalIndent(plannedCfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal planned XML: %v", err)
	}
	summary.PlannedConfig = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n" + string(xmlContent)

	// 2. Check SSH and read current config
	var currentConfig string
	path := SoundTouchSdkPrivateCfgPath

	// Check file details
	fileInfo, _ := client.Run(fmt.Sprintf("ls -l %s", path))
	if fileInfo != "" {
		fmt.Printf("File info for %s: %s\n", path, fileInfo)
	}

	// Try cat
	config, err := client.Run(fmt.Sprintf("cat %s", path))
	if err == nil && config != "" {
		currentConfig = config
		summary.SSHSuccess = true
		summary.CurrentConfig = currentConfig
		fmt.Printf("Current config from %s at %s (length: %d):\n%q\n", deviceIP, path, len(currentConfig), currentConfig)
	} else {
		// Fallback: try base64 if cat returned empty string but file has size > 0
		if config == "" && fileInfo != "" {
			fmt.Printf("Cat returned empty for %s, trying base64\n", path)
			b64Config, err := client.Run(fmt.Sprintf("base64 %s", path))
			if err == nil && b64Config != "" {
				fmt.Printf("Base64 output for %s (length %d)\n", path, len(b64Config))
				// We don't decode it yet, just reporting it might be there.
				// But for the summary, let's just use what cat (or lack thereof) returned.
			}
		}

		// If SSH failed or file couldn't be read
		if _, sshErr := client.Run("ls /"); sshErr == nil {
			summary.SSHSuccess = true
			if err != nil {
				summary.CurrentConfig = fmt.Sprintf("Error reading config: %v", err)
			} else {
				summary.CurrentConfig = config // Might be empty
			}
		} else {
			summary.SSHSuccess = false
			summary.CurrentConfig = fmt.Sprintf("SSH connection failed: %v", sshErr)
		}
	}

	// 3. Check for remote services files
	locations := []string{
		"/etc/remote_services",
		"/mnt/nv/remote_services",
		"/tmp/remote_services",
	}

	for _, loc := range locations {
		_, err := client.Run(fmt.Sprintf("[ -e %s ]", loc))
		if err == nil {
			summary.RemoteServicesFound = append(summary.RemoteServicesFound, loc)
			summary.RemoteServicesEnabled = true
			if loc != "/tmp/remote_services" {
				summary.RemoteServicesPersistent = true
			}
		}
	}

	return summary, nil
}

// MigrateSpeaker configures the speaker at the given IP to use this soundcork service.
func (m *Manager) MigrateSpeaker(deviceIP string, targetURL string) error {
	if targetURL == "" {
		targetURL = m.ServerURL
	}
	if err := m.EnsureRemoteServices(deviceIP); err != nil {
		// Log but continue migration? Or fail? The requirement is "to ensure stable 'remote_services'"
		// Let's log it.
		fmt.Printf("Warning: failed to ensure remote services: %v\n", err)
	}

	cfg := PrivateCfg{
		MargeServerUrl:             fmt.Sprintf("%s/marge", targetURL),
		StatsServerUrl:             targetURL,
		SwUpdateUrl:                fmt.Sprintf("%s/updates/soundtouch", targetURL),
		UsePandoraProductionServer: true,
		IsZeroconfEnabled:          true,
		SaveMargeCustomerReport:    false,
		BmxRegistryUrl:             fmt.Sprintf("%s/bmx/registry/v1/services", targetURL),
	}

	xmlContent, err := xml.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}

	// Add XML header
	xmlContent = append([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"), xmlContent...)

	client := ssh.NewClient(deviceIP)

	// 1. Upload the configuration
	remotePath := SoundTouchSdkPrivateCfgPath
	if err := client.UploadContent(xmlContent, remotePath); err != nil {
		return fmt.Errorf("failed to upload config: %v", err)
	}

	// 2. Reboot the speaker (requires 'rw' command first to make filesystem writable)
	if _, err := client.Run("rw && reboot"); err != nil {
		return fmt.Errorf("failed to reboot speaker: %v", err)
	}

	return nil
}

// EnsureRemoteServices ensures that remote services are enabled on the device.
// It tries to create an empty file in one of the known valid locations.
func (m *Manager) EnsureRemoteServices(deviceIP string) error {
	client := ssh.NewClient(deviceIP)

	// Try locations in order of preference
	locations := []string{
		"/etc/remote_services",
		"/mnt/nv/remote_services",
		"/tmp/remote_services",
	}

	// First, try to make the filesystem writable
	// Many SoundTouch devices have read-only root filesystems
	_, _ = client.Run("rw")

	for _, loc := range locations {
		_, err := client.Run(fmt.Sprintf("touch %s", loc))
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("failed to enable remote services in any of the locations: %v", locations)
}
