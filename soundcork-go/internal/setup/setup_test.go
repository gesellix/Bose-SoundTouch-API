package setup

import (
	"fmt"
	"testing"
)

func TestGetMigrationSummary_SSHFailure(t *testing.T) {
	// Use an IP that is unlikely to have an SSH server running or reachable
	// or use a local port that is closed.
	// We'll use a local port that we know is closed.
	manager := NewManager("http://localhost:8000")
	summary, err := manager.GetMigrationSummary("127.0.0.1")

	// Currently it might return an error OR it might return a summary with SSHSuccess: false
	// but the issue description says the user is told connection SUCCEEDED.

	if err == nil {
		if summary.SSHSuccess {
			t.Errorf("Expected SSHSuccess to be false for closed port, got true")
		}
		if summary.CurrentConfig == "" {
			t.Errorf("Expected CurrentConfig to contain error message, got empty string")
		}
		fmt.Printf("Got expected SSH failure summary: %s\n", summary.CurrentConfig)
	} else {
		t.Errorf("Expected no error from GetMigrationSummary, got %v", err)
	}
}
