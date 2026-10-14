package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type Config struct {
	mu         sync.RWMutex
	ConfigPath string         `yaml:"-"`
	Server     ServerConfig   `yaml:"server"`
	Oxidized   OxidizedConfig `yaml:"oxidized"`
	Auth       AuthConfig     `yaml:"auth"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type OxidizedConfig struct {
	DBPath         string `yaml:"db_path"`
	RestartCommand string `yaml:"restart_command"`
	StatusCommand  string `yaml:"status_command"`
	AutoBackup     bool   `yaml:"auto_backup"`
}

type AuthConfig struct {
	Username      string `yaml:"username"`
	PasswordHash  string `yaml:"password_hash"`
	Password      string `yaml:"password,omitempty"` // For initial easy setup
	SessionSecret string `yaml:"session_secret"`
}

// DefaultConfig returns safe default settings
func DefaultConfig() *Config {
	// Default password is "admin"
	defaultHash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	secret := generateRandomSecret(32)

	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Oxidized: OxidizedConfig{
			DBPath:         "./router.db",
			RestartCommand: "systemctl restart oxidized.service",
			StatusCommand:  "systemctl is-active oxidized.service",
			AutoBackup:     true,
		},
		Auth: AuthConfig{
			Username:      "admin",
			PasswordHash:  string(defaultHash),
			SessionSecret: secret,
		},
	}
}

// Load loads configuration from the given file path
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	cfg.ConfigPath = path

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Create directory if not exists
			dir := filepath.Dir(path)
			if dir != "." && dir != "" {
				_ = os.MkdirAll(dir, 0755)
			}
			// Save default config
			if saveErr := cfg.Save(); saveErr != nil {
				return nil, fmt.Errorf("failed to create default config file %s: %w", path, saveErr)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config %s: %w", path, err)
	}

	cfg.ConfigPath = path

	// If plaintext password is provided in config instead of hash, convert it to hash
	var needsSave bool
	if cfg.Auth.Password != "" && cfg.Auth.PasswordHash == "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Auth.Password), bcrypt.DefaultCost)
		if err == nil {
			cfg.Auth.PasswordHash = string(hash)
			cfg.Auth.Password = ""
			needsSave = true
		}
	}

	if cfg.Auth.SessionSecret == "" {
		cfg.Auth.SessionSecret = generateRandomSecret(32)
		needsSave = true
	}

	if needsSave {
		_ = cfg.Save()
	}

	return cfg, nil
}

// Save writes the current configuration to disk
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write atomically or directly
	if err := os.WriteFile(c.ConfigPath, data, 0640); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", c.ConfigPath, err)
	}

	return nil
}

// VerifyPassword checks if the provided password matches the configured hash or plaintext
func (c *Config) VerifyPassword(password string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.Auth.PasswordHash != "" {
		err := bcrypt.CompareHashAndPassword([]byte(c.Auth.PasswordHash), []byte(password))
		return err == nil
	}

	// Fallback if only plaintext password was specified
	if c.Auth.Password != "" && c.Auth.Password == password {
		return true
	}

	return false
}

// UpdatePassword updates the stored password with a new bcrypt hash and writes to config
func (c *Config) UpdatePassword(newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	c.mu.Lock()
	c.Auth.PasswordHash = string(hash)
	c.Auth.Password = "" // remove plaintext if present
	c.mu.Unlock()

	return c.Save()
}

// GetUsername returns current username safely
func (c *Config) GetUsername() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Auth.Username
}

// GetDBPath returns current database path safely
func (c *Config) GetDBPath() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.Oxidized.DBPath
}

// GetPort returns current server port safely
func (c *Config) GetPort() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.Server.Port <= 0 {
		return 8080
	}
	return c.Server.Port
}

func generateRandomSecret(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "oxidized-manager-secret-token-key-2026"
	}
	return hex.EncodeToString(bytes)
}
