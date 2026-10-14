package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"oxidized_edit/internal/auth"
	"oxidized_edit/internal/config"
	"oxidized_edit/internal/database"
	"oxidized_edit/internal/handlers"
	"oxidized_edit/internal/service"
	"oxidized_edit/web"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	log.Printf("[INFO] Loading configuration from %s...", *configPath)
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[FATAL] Could not load config: %v", err)
	}

	// If default config was just created or hash was missing, ensure admin password works
	if cfg.Auth.PasswordHash == "" && cfg.Auth.Password == "" {
		_ = cfg.UpdatePassword("admin")
	}

	// Initialize Router DB
	dbPath := cfg.GetDBPath()
	log.Printf("[INFO] Initializing Oxidized router.db manager for: %s", dbPath)
	routerDB := database.NewRouterDB(dbPath, cfg.Oxidized.AutoBackup)
	if err := routerDB.Load(); err != nil {
		log.Printf("[WARN] Error loading router.db (%v). It will be created on save.", err)
	}

	// Initialize Oxidized Service Manager
	oxidizedSvc := service.NewOxidizedManager(cfg.Oxidized.RestartCommand, cfg.Oxidized.StatusCommand)

	// Initialize Auth Sessions
	sessionStore := auth.NewSessionStore(24 * time.Hour)

	// Initialize API Handlers
	api := handlers.NewAPI(cfg, routerDB, oxidizedSvc, sessionStore)

	// Setup HTTP Router
	mux := http.NewServeMux()

	// Embedded Static Assets
	staticFS, err := web.GetFileSystem()
	if err != nil {
		log.Fatalf("[FATAL] Failed to load embedded web assets: %v", err)
	}
	fileServer := http.FileServer(staticFS)
	mux.Handle("/", fileServer)

	// Public Auth Endpoints
	mux.HandleFunc("/api/login", api.LoginHandler)
	mux.HandleFunc("/api/logout", api.LogoutHandler)
	mux.HandleFunc("/api/auth/me", api.MeHandler)

	// Protected Endpoints
	mux.HandleFunc("/api/auth/change-password", api.AuthMiddleware(api.ChangePasswordHandler))

	// Device Endpoints
	mux.HandleFunc("/api/devices", api.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetDevicesHandler(w, r)
		case http.MethodPost:
			api.AddDeviceHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/devices/", api.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			api.UpdateDeviceHandler(w, r)
		case http.MethodDelete:
			api.DeleteDeviceHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/devices/raw", api.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetRawHandler(w, r)
		case http.MethodPut, http.MethodPost:
			api.SaveRawHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/devices/reload", api.AuthMiddleware(api.ReloadDBHandler))

	// Oxidized Service Endpoints
	mux.HandleFunc("/api/oxidized/restart", api.AuthMiddleware(api.RestartOxidizedHandler))
	mux.HandleFunc("/api/oxidized/status", api.AuthMiddleware(api.OxidizedStatusHandler))

	// Server Address
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.GetPort())
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 25 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server banner
	fmt.Println("================================================================")
	fmt.Println("       🚀 OXIDIZED ROUTER.DB WEB MANAGER")
	fmt.Println("================================================================")
	fmt.Printf(" [✓] Listening on:     http://%s\n", addr)
	if cfg.Server.Host == "0.0.0.0" {
		fmt.Printf(" [✓] Local URL:        http://localhost:%d\n", cfg.GetPort())
	}
	fmt.Printf(" [✓] Database File:    %s\n", dbPath)
	fmt.Printf(" [✓] Restart Command:  %s\n", cfg.Oxidized.RestartCommand)
	fmt.Printf(" [✓] Web Console Port: %d (configured in %s)\n", cfg.GetPort(), *configPath)
	fmt.Println("----------------------------------------------------------------")
	fmt.Printf(" Default login: %s / admin\n", cfg.GetUsername())
	fmt.Println("================================================================")

	// Graceful shutdown handling
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Server error: %v", err)
		}
	}()

	<-stopChan
	log.Println("[INFO] Shutting down Oxidized Web Manager gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[ERROR] Server forced to shutdown: %v", err)
	}

	log.Println("[INFO] Server exited successfully.")
}
