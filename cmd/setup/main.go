package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mash/internal/config"
)

func main() {
	configPath := "config.json"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	err := RunSetup(os.Stdin, os.Stdout, configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error during setup: %v\n", err)
		os.Exit(1)
	}
}

// RunSetup executes the interactive setup wizard for MASH.
func RunSetup(in io.Reader, out io.Writer, configPath string) error {
	reader := bufio.NewReader(in)

	fmt.Fprintln(out, "=====================================================================")
	fmt.Fprintln(out, "                    MAS-H Interactive Setup Tool                     ")
	fmt.Fprintln(out, "=====================================================================")
	fmt.Fprintln(out, "This script is like rclone setup. It is completely NON-DESTRUCTIVE.")
	fmt.Fprintln(out, "It registers your storage provider (drive/folder) to the MAS-H system")
	fmt.Fprintln(out, "without overwriting, modifying, or deleting any of your existing files.")
	fmt.Fprintln(out, "=====================================================================")
	fmt.Fprintln(out)

	// 1. Load or initialize config
	var cfg config.Config
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fmt.Errorf("error reading config file: %w", err)
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("error parsing config file: %w", err)
		}
	} else if os.IsNotExist(err) {
		fmt.Fprintln(out, "[Info] Config file not found. Creating a new one.")
		cfg = config.Config{
			Port:     8080,
			DBPath:   "mash.db",
			Services: []string{"tuoni", "seshat", "observer", "caretaker", "boatman", "gateway"},
			Providers: []config.StorageProvider{},
			Policies:  []config.Policy{},
		}
	} else {
		return fmt.Errorf("error checking config file: %w", err)
	}

	// 2. Ask for drive/folder path
	var path string
	for {
		path = promptString(reader, out, "Enter the path of the drive or folder to allocate", "")
		if path != "" {
			break
		}
		fmt.Fprintln(out, "Path cannot be empty. Please enter a valid path.")
	}

	// Make path absolute/clean to be robust
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}

	// Offer to create directory if it does not exist
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(out, "Directory '%s' does not exist.\n", path)
		if promptYesNo(reader, out, "Would you like MAS-H to create this folder?", true) {
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			fmt.Fprintln(out, "Directory created successfully.")
		} else {
			fmt.Fprintln(out, "[Warning] Allocated path does not exist on disk yet.")
		}
	}

	// 3. Ask for unique provider ID
	defaultID := filepath.Base(path)
	if defaultID == "." || defaultID == "/" || defaultID == "" {
		defaultID = "allocated-drive"
	}
	providerID := promptString(reader, out, "Enter a unique ID for this storage provider", defaultID)

	// Check for unique ID collision
	for _, p := range cfg.Providers {
		if p.ID == providerID {
			providerID = providerID + "-new"
			fmt.Fprintf(out, "[Notice] ID collision detected. Automatically renamed to: %s\n", providerID)
		}
	}

	// 4. Ask for drive type (SSD or HDD)
	driveTypeChoice := promptOptions(reader, out, "What is the physical media type of this storage?", []string{"SSD (high performance, low latency)", "HDD (high capacity, high latency)"}, 0)
	isSSD := driveTypeChoice == 0

	// 5. Confirm addition
	fmt.Fprintf(out, "\nReady to add provider:\n")
	fmt.Fprintf(out, "  ID:   %s\n", providerID)
	fmt.Fprintf(out, "  Path: %s\n", path)
	if isSSD {
		fmt.Fprintf(out, "  Type: SSD\n")
	} else {
		fmt.Fprintf(out, "  Type: HDD\n")
	}

	if !promptYesNo(reader, out, "Confirm allocation of this space?", true) {
		fmt.Fprintln(out, "Setup aborted. No changes made.")
		return nil
	}

	// 6. Ask what level of control we want to give that allocated space
	fmt.Fprintln(out, "\nWhat level of control do you want to give that allocated space?")
	controlOptions := []string{
		"Only index and observe (Read-only, default. MAS-H scans but never replicates files TO this space)",
		"Full control (Read/Write. MAS-H can replicate or migrate files to this space)",
	}
	controlChoice := promptOptions(reader, out, "Enter choice (1 or 2)", controlOptions, 0)

	// Create Storage Provider object
	newProvider := config.StorageProvider{
		ID:           providerID,
		Type:         "local",
		Path:         path,
		Capabilities: make(map[string]string),
	}

	if isSSD {
		newProvider.Capabilities["latency"] = "low"
		newProvider.Capabilities["drive_type"] = "ssd"
	} else {
		newProvider.Capabilities["latency"] = "high"
		newProvider.Capabilities["drive_type"] = "hdd"
	}

	if controlChoice == 0 {
		newProvider.Capabilities["control"] = "index_observe"
		newProvider.Capabilities["read_only"] = "true"
	} else {
		newProvider.Capabilities["control"] = "full"
		newProvider.Capabilities["read_only"] = "false"
	}

	// Add to existing providers list
	cfg.Providers = append(cfg.Providers, newProvider)
	fmt.Fprintf(out, "Successfully registered storage provider '%s'.\n", providerID)

	// 7. Check if there are a mix of drives (SSDs and HDDs)
	var hasSSD, hasHDD bool
	for _, p := range cfg.Providers {
		latency := ""
		if p.Capabilities != nil {
			latency = p.Capabilities["latency"]
		}
		isP_SSD := latency == "low" || (p.Capabilities != nil && p.Capabilities["drive_type"] == "ssd")
		isP_HDD := latency == "high" || (p.Capabilities != nil && p.Capabilities["drive_type"] == "hdd")
		if isP_SSD {
			hasSSD = true
		}
		if isP_HDD {
			hasHDD = true
		}
	}

	if hasSSD && hasHDD {
		fmt.Fprintln(out, "\n[Mix Detected] System noticed a mix of storage media (SSDs and HDDs) in your configuration.")
		fmt.Fprintln(out, "MAS-H can automatically move files around according to its storage tiering logic:")
		fmt.Fprintln(out, "  - Hot/frequently accessed files are placed on fast SSD storage.")
		fmt.Fprintln(out, "  - Cold/infrequently accessed files are moved to high-capacity HDD storage.")
		if promptYesNo(reader, out, "Would you like to enable automated file migration based on access frequency?", true) {
			// Check if migrate policy already exists
			hasMigratePolicy := false
			for _, policy := range cfg.Policies {
				if policy.Type == "migrate" {
					hasMigratePolicy = true
					break
				}
			}
			if !hasMigratePolicy {
				migratePolicy := config.Policy{
					ID:       "tier-by-frequency",
					Type:     "migrate",
					Target:   "object",
					Value:    "ssd-for-frequent-hdd-for-infrequent",
					Priority: 10,
				}
				cfg.Policies = append(cfg.Policies, migratePolicy)
				fmt.Fprintln(out, "Automated file migration/tiering rule enabled.")
			} else {
				fmt.Fprintln(out, "Automated file migration/tiering rule is already active.")
			}
		}
	}

	// 8. Save config
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format config: %w", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	fmt.Fprintf(out, "\nSetup completed successfully! Config saved to: %s\n", configPath)
	return nil
}

func promptString(reader *bufio.Reader, out io.Writer, msg string, defaultVal string) string {
	if defaultVal != "" {
		fmt.Fprintf(out, "%s [%s]: ", msg, defaultVal)
	} else {
		fmt.Fprintf(out, "%s: ", msg)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptYesNo(reader *bufio.Reader, out io.Writer, msg string, defaultYes bool) bool {
	var options string
	if defaultYes {
		options = "Y/n"
	} else {
		options = "y/N"
	}
	fmt.Fprintf(out, "%s [%s]: ", msg, options)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	if input == "" {
		return defaultYes
	}
	if input == "y" || input == "yes" {
		return true
	}
	return false
}

func promptOptions(reader *bufio.Reader, out io.Writer, msg string, options []string, defaultIdx int) int {
	for i, opt := range options {
		fmt.Fprintf(out, "  %d) %s\n", i+1, opt)
	}
	defaultStr := fmt.Sprintf("%d", defaultIdx+1)
	fmt.Fprintf(out, "%s [%s]: ", msg, defaultStr)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultIdx
	}
	val, err := strconv.Atoi(input)
	if err != nil || val < 1 || val > len(options) {
		fmt.Fprintf(out, "Invalid choice. Defaulting to choice %s\n", defaultStr)
		return defaultIdx
	}
	return val - 1
}
