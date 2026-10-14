package config

import (
	"os"
	"testing"
)

func TestConfigLoadAndVerifyPassword(t *testing.T) {
	tmpFile := "test_config.yaml"
	defer os.Remove(tmpFile)

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create default config: %v", err)
	}

	if !cfg.VerifyPassword("admin") {
		t.Errorf("Expected default password 'admin' to verify, but it failed")
	}

	err = cfg.UpdatePassword("newSecretPass123")
	if err != nil {
		t.Fatalf("Failed to update password: %v", err)
	}

	if cfg.VerifyPassword("admin") {
		t.Errorf("Expected old password 'admin' to fail, but it verified")
	}

	if !cfg.VerifyPassword("newSecretPass123") {
		t.Errorf("Expected new password to verify")
	}

	// Re-load from disk to verify persistence
	cfg2, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	if !cfg2.VerifyPassword("newSecretPass123") {
		t.Errorf("Expected reloaded config to verify new password")
	}
}
