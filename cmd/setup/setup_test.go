package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mash/internal/config"
)

func TestRunSetup_IndexObserveSSD(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	drivePath := filepath.Join(tmpDir, "my-ssd-drive")

	// Set up simulated inputs
	inputs := []string{
		drivePath,    // Enter path
		"y",          // Would you like MAS-H to create this folder? (Y/n)
		"test-ssd",   // Unique ID
		"1",          // Drive type SSD (Choice 1)
		"y",          // Confirm allocation? (Y/n)
		"1",          // Control level: Only index and observe (Choice 1)
	}
	inputStr := strings.Join(inputs, "\n") + "\n"
	in := strings.NewReader(inputStr)
	var out bytes.Buffer

	err := RunSetup(in, &out, configPath)
	if err != nil {
		t.Fatalf("RunSetup failed: %v", err)
	}

	// Read saved config to verify
	cfgData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if len(cfg.Providers) != 1 {
		t.Fatalf("Expected 1 provider, got %d", len(cfg.Providers))
	}

	p := cfg.Providers[0]
	if p.ID != "test-ssd" {
		t.Errorf("Expected provider ID 'test-ssd', got '%s'", p.ID)
	}
	if p.Capabilities["latency"] != "low" || p.Capabilities["drive_type"] != "ssd" {
		t.Errorf("Expected low latency ssd provider capabilities, got %+v", p.Capabilities)
	}
	if p.Capabilities["control"] != "index_observe" || p.Capabilities["read_only"] != "true" {
		t.Errorf("Expected index_observe control, got %+v", p.Capabilities)
	}
}

func TestRunSetup_MixAndTiering(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	ssdPath := filepath.Join(tmpDir, "my-ssd")
	hddPath := filepath.Join(tmpDir, "my-hdd")

	// 1. First add SSD
	inputsSSD := []string{
		ssdPath,
		"y",
		"provider-ssd",
		"1", // SSD
		"y", // confirm
		"2", // Full control
	}
	inSSD := strings.NewReader(strings.Join(inputsSSD, "\n") + "\n")
	var outSSD bytes.Buffer
	if err := RunSetup(inSSD, &outSSD, configPath); err != nil {
		t.Fatalf("Failed to add first provider: %v", err)
	}

	// 2. Add HDD (which will trigger mix detection)
	inputsHDD := []string{
		hddPath,
		"y",
		"provider-hdd",
		"2", // HDD
		"y", // confirm
		"2", // Full control
		"y", // Enable automated migration (Y/n)
	}
	inHDD := strings.NewReader(strings.Join(inputsHDD, "\n") + "\n")
	var outHDD bytes.Buffer
	if err := RunSetup(inHDD, &outHDD, configPath); err != nil {
		t.Fatalf("Failed to add second provider: %v", err)
	}

	// Read saved config to verify
	cfgData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var cfg config.Config
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		t.Fatalf("Failed to unmarshal saved config: %v", err)
	}

	if len(cfg.Providers) != 2 {
		t.Fatalf("Expected 2 providers, got %d", len(cfg.Providers))
	}

	// Check if migrate policy has been added
	var foundMigrate bool
	for _, policy := range cfg.Policies {
		if policy.Type == "migrate" {
			foundMigrate = true
			if policy.ID != "tier-by-frequency" {
				t.Errorf("Expected policy ID 'tier-by-frequency', got '%s'", policy.ID)
			}
		}
	}
	if !foundMigrate {
		t.Errorf("Expected automated file migration/tiering policy to be enabled")
	}
}
