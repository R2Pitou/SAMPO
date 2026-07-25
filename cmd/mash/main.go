package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mash/internal/config"
	"mash/internal/event"
	"mash/internal/gateway"
	"mash/internal/librarian"
	"mash/internal/staff"
)

func main() {
	configPath := flag.String("config", "config.json", "path to configuration file")
	flag.Parse()

	log.Println("[MASH] Launching Memory Abstraction Storage Hypervisor...")

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Printf("[MASH] Warning: Could not load config.json (%v). Using default fallback configuration.", err)
		cfg = &config.Config{
			Port:     8080,
			Services: []string{"tuoni", "seshat", "observer", "caretaker", "boatman", "gateway"},
		}
	}

	// 1. Initialize core channels and Event Bus
	eventBus := event.NewBus()

	// 2. Initialize Optional SQLite Database Store
	var dbStore *librarian.SQLiteStore
	if cfg.DBPath != "" {
		store, err := librarian.NewSQLiteStore(cfg.DBPath)
		if err != nil {
			log.Fatalf("[MASH] Failed to initialize SQLite storage at %s: %v", cfg.DBPath, err)
		}
		dbStore = store
		log.Printf("[MASH] Persistent SQLite database connected: %s", cfg.DBPath)
	}

	// 3. Initialize Seshat Ledger with optional SQLite persistence
	catalog := librarian.NewSeshat(eventBus, dbStore)

	// Register Storage Providers from Config
	for _, provider := range cfg.Providers {
		p := provider
		catalog.AddProvider(&p)
		log.Printf("[MASH] Registered Storage Provider %s (%s) path: %s", p.ID, p.Type, p.Path)
	}

	// 4. Launch Staff Members depending on Config Services
	var activeObserver *staff.Observer
	var activeCaretaker *staff.Caretaker

	for _, s := range cfg.Services {
		switch s {
		case "tuoni":
			_ = staff.NewTuoni(catalog, eventBus, cfg.Policies)
			log.Println("[MASH] Tuoni (Reasoning Engine) started.")
		case "boatman":
			_ = staff.NewBoatman(catalog, eventBus)
			log.Println("[MASH] Boatman (Transfer Engine) started.")
		case "observer":
			activeObserver = staff.NewObserver(catalog, eventBus)
			activeObserver.Start(2 * time.Second)
			log.Println("[MASH] Observer daemon started.")
		case "caretaker":
			activeCaretaker = staff.NewCaretaker(catalog, eventBus)
			activeCaretaker.Start(5 * time.Second)
			log.Println("[MASH] Caretaker maintenance daemon started.")
		case "gateway":
			gw := gateway.NewGateway(catalog, cfg.Port)
			go func() {
				log.Printf("[MASH] Gateway REST server running on port %d...", cfg.Port)
				if err := gw.Start(); err != nil {
					log.Printf("[MASH] Gateway HTTP server error: %v", err)
				}
			}()
		}
	}

	log.Println("[MASH] Control plane completely online.")

	// Wait for OS interruption to gracefully terminate
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("[MASH] Shutting down...")
	if activeObserver != nil {
		activeObserver.Stop()
	}
	if activeCaretaker != nil {
		activeCaretaker.Stop()
	}
	if dbStore != nil {
		if err := dbStore.Close(); err != nil {
			log.Printf("[MASH] Error closing database: %v", err)
		} else {
			log.Println("[MASH] Persistent database disconnected cleanly.")
		}
	}
	log.Println("[MASH] Off-line cleanly.")
}
