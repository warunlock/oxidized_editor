package database

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Device represents a single line in oxidized router.db
// Format: name:ip:model:username:password:group:port
type Device struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Model    string `json:"model"`
	Username string `json:"username"`
	Password string `json:"password"`
	Group    string `json:"group"`
	Port     string `json:"port"`
}

// RouterDB manages reading, writing, and modifying router.db
type RouterDB struct {
	mu         sync.RWMutex
	filePath   string
	autoBackup bool
	devices    []Device
	comments   []string
	lastMod    time.Time
}

// NewRouterDB creates a new RouterDB manager
func NewRouterDB(filePath string, autoBackup bool) *RouterDB {
	return &RouterDB{
		filePath:   filePath,
		autoBackup: autoBackup,
		devices:    make([]Device, 0),
		comments:   make([]string, 0),
	}
}

// Load reads and parses the router.db file from disk
func (r *RouterDB) Load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.loadUnlocked()
}

func (r *RouterDB) loadUnlocked() error {
	fileInfo, err := os.Stat(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Ensure parent dir exists
			dir := filepath.Dir(r.filePath)
			if dir != "." && dir != "" {
				_ = os.MkdirAll(dir, 0755)
			}
			// Create empty file with header comment
			initialContent := "# Oxidized router.db\n# Format: name:ip:model:username:password:group:port\n"
			if writeErr := os.WriteFile(r.filePath, []byte(initialContent), 0644); writeErr != nil {
				return fmt.Errorf("failed to create initial router.db at %s: %w", r.filePath, writeErr)
			}
			fileInfo, _ = os.Stat(r.filePath)
		} else {
			return fmt.Errorf("failed to stat %s: %w", r.filePath, err)
		}
	}

	file, err := os.Open(r.filePath)
	if err != nil {
		return fmt.Errorf("failed to open router.db at %s: %w", r.filePath, err)
	}
	defer file.Close()

	var devices []Device
	var comments []string

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			comments = append(comments, line)
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			// Ignore or keep as comment if corrupt
			comments = append(comments, fmt.Sprintf("# [malformed line %d]: %s", lineNum, line))
			continue
		}

		dev := Device{
			ID:    generateDeviceID(),
			Name:  strings.TrimSpace(parts[0]),
			IP:    strings.TrimSpace(parts[1]),
			Model: strings.TrimSpace(parts[2]),
		}

		if len(parts) > 3 {
			dev.Username = strings.TrimSpace(parts[3])
		}
		if len(parts) > 4 {
			dev.Password = strings.TrimSpace(parts[4])
		}
		if len(parts) > 5 {
			dev.Group = strings.TrimSpace(parts[5])
		}
		if len(parts) > 6 {
			dev.Port = strings.TrimSpace(parts[6])
		}

		devices = append(devices, dev)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading router.db: %w", err)
	}

	r.devices = devices
	r.comments = comments
	if fileInfo != nil {
		r.lastMod = fileInfo.ModTime()
	} else {
		r.lastMod = time.Now()
	}

	return nil
}

// GetAll returns a copy of all devices
func (r *RouterDB) GetAll() ([]Device, time.Time, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Device, len(r.devices))
	copy(result, r.devices)
	return result, r.lastMod, r.filePath
}

// GetByID finds device by ID
func (r *RouterDB) GetByID(id string) (Device, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, dev := range r.devices {
		if dev.ID == id {
			return dev, true
		}
	}
	return Device{}, false
}

// Add adds a new device
func (r *RouterDB) Add(dev *Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := validateDevice(dev); err != nil {
		return err
	}

	// Check duplicate name
	for _, existing := range r.devices {
		if strings.EqualFold(existing.Name, dev.Name) {
			return fmt.Errorf("device with name '%s' already exists", dev.Name)
		}
	}

	if dev.ID == "" {
		dev.ID = generateDeviceID()
	}

	r.devices = append(r.devices, *dev)
	return r.saveUnlocked()
}

// Update updates an existing device
func (r *RouterDB) Update(id string, updated Device) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := validateDevice(&updated); err != nil {
		return err
	}

	index := -1
	for i, dev := range r.devices {
		if dev.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("device with ID '%s' not found", id)
	}

	// Check name uniqueness among other devices
	for i, existing := range r.devices {
		if i != index && strings.EqualFold(existing.Name, updated.Name) {
			return fmt.Errorf("another device with name '%s' already exists", updated.Name)
		}
	}

	updated.ID = id
	r.devices[index] = updated
	return r.saveUnlocked()
}

// Delete removes a device by ID
func (r *RouterDB) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := -1
	for i, dev := range r.devices {
		if dev.ID == id {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("device with ID '%s' not found", id)
	}

	r.devices = append(r.devices[:index], r.devices[index+1:]...)
	return r.saveUnlocked()
}

// GetRaw returns the full raw text content of router.db
func (r *RouterDB) GetRaw() (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveRaw overwrites the router.db file with raw text, then reloads
func (r *RouterDB) SaveRaw(rawContent string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Backup before overwriting
	if r.autoBackup {
		_ = r.createBackupUnlocked()
	}

	if err := os.WriteFile(r.filePath, []byte(rawContent), 0644); err != nil {
		return fmt.Errorf("failed to write raw content to %s: %w", r.filePath, err)
	}

	return r.loadUnlocked()
}

// saveUnlocked writes the devices slice back to router.db format
func (r *RouterDB) saveUnlocked() error {
	if r.autoBackup {
		_ = r.createBackupUnlocked()
	}

	var sb strings.Builder

	// Header comments
	if len(r.comments) > 0 {
		for _, c := range r.comments {
			sb.WriteString(c)
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("# Oxidized router.db\n# Format: name:ip:model:username:password:group:port\n")
	}

	for _, dev := range r.devices {
		// name:ip:model:username:password:group:port
		line := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%s\n",
			cleanField(dev.Name),
			cleanField(dev.IP),
			cleanField(dev.Model),
			cleanField(dev.Username),
			cleanField(dev.Password),
			cleanField(dev.Group),
			cleanField(dev.Port),
		)
		sb.WriteString(line)
	}

	tempFile := r.filePath + ".tmp"
	if err := os.WriteFile(tempFile, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	if err := os.Rename(tempFile, r.filePath); err != nil {
		// Fallback direct write on Windows if rename fails due to lock
		if writeErr := os.WriteFile(r.filePath, []byte(sb.String()), 0644); writeErr != nil {
			return fmt.Errorf("failed to save router.db: %w", writeErr)
		}
		_ = os.Remove(tempFile)
	}

	r.lastMod = time.Now()
	return nil
}

func (r *RouterDB) createBackupUnlocked() error {
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return err
	}

	bakPath := r.filePath + ".bak"
	return os.WriteFile(bakPath, data, 0644)
}

func cleanField(val string) string {
	// Oxidized fields cannot contain colons or newlines
	s := strings.TrimSpace(val)
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

func validateDevice(dev *Device) error {
	dev.Name = strings.TrimSpace(dev.Name)
	dev.IP = strings.TrimSpace(dev.IP)
	dev.Model = strings.TrimSpace(dev.Model)
	dev.Username = strings.TrimSpace(dev.Username)
	dev.Password = strings.TrimSpace(dev.Password)
	dev.Group = strings.TrimSpace(dev.Group)
	dev.Port = strings.TrimSpace(dev.Port)

	if dev.Name == "" {
		return fmt.Errorf("device name is required")
	}
	if dev.IP == "" {
		return fmt.Errorf("device IP / hostname is required")
	}
	if dev.Model == "" {
		return fmt.Errorf("device model is required")
	}
	if strings.Contains(dev.Name, ":") {
		return fmt.Errorf("device name cannot contain colon ':'")
	}
	if strings.Contains(dev.IP, ":") {
		return fmt.Errorf("device IP cannot contain colon ':'")
	}
	return nil
}

func generateDeviceID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("dev-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b))
}
