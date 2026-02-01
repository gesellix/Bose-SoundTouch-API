package setup

import (
	"encoding/xml"
	"fmt"

	"github.com/deborahgu/soundcork/internal/ssh"
)

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
	SSHSuccess    bool   `json:"ssh_success"`
	CurrentConfig string `json:"current_config"`
	PlannedConfig string `json:"planned_config"`
}

// Manager handles the migration of speakers to the soundcork service.
type Manager struct {
	ServerURL string
}

// NewManager creates a new Manager with the given base server URL.
func NewManager(serverURL string) *Manager {
	return &Manager{ServerURL: serverURL}
}

// GetMigrationSummary returns a summary of the current and planned state of the speaker.
func (m *Manager) GetMigrationSummary(deviceIP string) (*MigrationSummary, error) {
	client := ssh.NewClient(deviceIP)

	summary := &MigrationSummary{
		SSHSuccess: false,
	}

	// 1. Generate planned config (we can do this anyway)
	plannedCfg := PrivateCfg{
		MargeServerUrl:             fmt.Sprintf("%s/marge", m.ServerURL),
		StatsServerUrl:             m.ServerURL,
		SwUpdateUrl:                fmt.Sprintf("%s/updates/soundtouch", m.ServerURL),
		UsePandoraProductionServer: true,
		IsZeroconfEnabled:          true,
		SaveMargeCustomerReport:    false,
		BmxRegistryUrl:             fmt.Sprintf("%s/bmx/registry/v1/services", m.ServerURL),
	}

	xmlContent, err := xml.MarshalIndent(plannedCfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal planned XML: %v", err)
	}
	summary.PlannedConfig = "<?xml version=\"1.0\" encoding=\"utf-8\"?>\n" + string(xmlContent)

	// 2. Check SSH and read current config
	remotePath := "/opt/Bose/etc/SoundTouchSdkPrivateCfg.xml"
	currentConfig, err := client.Run(fmt.Sprintf("cat %s", remotePath))
	if err != nil {
		// It's possible the file doesn't exist yet or SSH failed.
		// We'll report SSH success if we can at least run a simple command.
		if _, sshErr := client.Run("ls /"); sshErr == nil {
			summary.SSHSuccess = true
			summary.CurrentConfig = fmt.Sprintf("Error reading config: %v", err)
		} else {
			// Instead of returning an error, we return a summary with SSHSuccess = false
			summary.SSHSuccess = false
			summary.CurrentConfig = fmt.Sprintf("SSH connection failed: %v", sshErr)
		}
	} else {
		summary.SSHSuccess = true
		summary.CurrentConfig = currentConfig
	}

	return summary, nil
}

// MigrateSpeaker configures the speaker at the given IP to use this soundcork service.
func (m *Manager) MigrateSpeaker(deviceIP string) error {
	cfg := PrivateCfg{
		MargeServerUrl:             fmt.Sprintf("%s/marge", m.ServerURL),
		StatsServerUrl:             m.ServerURL,
		SwUpdateUrl:                fmt.Sprintf("%s/updates/soundtouch", m.ServerURL),
		UsePandoraProductionServer: true,
		IsZeroconfEnabled:          true,
		SaveMargeCustomerReport:    false,
		BmxRegistryUrl:             fmt.Sprintf("%s/bmx/registry/v1/services", m.ServerURL),
	}

	xmlContent, err := xml.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal XML: %v", err)
	}

	// Add XML header
	xmlContent = append([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"), xmlContent...)

	client := ssh.NewClient(deviceIP)

	// 1. Upload the configuration
	remotePath := "/opt/Bose/etc/SoundTouchSdkPrivateCfg.xml"
	if err := client.UploadContent(xmlContent, remotePath); err != nil {
		return fmt.Errorf("failed to upload config: %v", err)
	}

	// 2. Reboot the speaker (requires 'rw' command first to make filesystem writable)
	if _, err := client.Run("rw && reboot"); err != nil {
		return fmt.Errorf("failed to reboot speaker: %v", err)
	}

	return nil
}
