package database

import (
	"os"
	"strings"
	"testing"
)

func TestRouterDB_CRUD(t *testing.T) {
	tmpDB := "test_router.db"
	defer os.Remove(tmpDB)
	defer os.Remove(tmpDB + ".bak")

	initial := `# Oxidized sample
# Format: name:ip:model:username:password:group:port
sw1:192.168.1.1:cisco:admin:cisco123:core:22
rt1:10.0.0.1:junos:root:junos123:edge:
`
	if err := os.WriteFile(tmpDB, []byte(initial), 0644); err != nil {
		t.Fatalf("Failed to write initial db: %v", err)
	}

	db := NewRouterDB(tmpDB, true)
	if err := db.Load(); err != nil {
		t.Fatalf("Failed to load db: %v", err)
	}

	devices, _, _ := db.GetAll()
	if len(devices) != 2 {
		t.Fatalf("Expected 2 devices, got %d", len(devices))
	}

	if devices[0].Name != "sw1" || devices[0].Port != "22" {
		t.Errorf("Unexpected device 0: %+v", devices[0])
	}
	if devices[1].Name != "rt1" || devices[1].Port != "" {
		t.Errorf("Unexpected device 1: %+v", devices[1])
	}

	// Test Add
	newDev := Device{
		Name:     "sw-acc01",
		IP:       "172.16.1.10",
		Model:    "routeros",
		Username: "admin",
		Password: "mikrotikpass",
		Group:    "access",
		Port:     "2222",
	}
	if err := db.Add(&newDev); err != nil {
		t.Fatalf("Failed to add device: %v", err)
	}

	devices, _, _ = db.GetAll()
	if len(devices) != 3 {
		t.Fatalf("Expected 3 devices after add, got %d", len(devices))
	}

	// Test Update
	addedID := devices[2].ID
	updateDev := newDev
	updateDev.IP = "172.16.1.20"
	updateDev.Group = "edge"
	if err := db.Update(addedID, updateDev); err != nil {
		t.Fatalf("Failed to update device: %v", err)
	}

	found, ok := db.GetByID(addedID)
	if !ok || found.IP != "172.16.1.20" || found.Group != "edge" {
		t.Errorf("Updated device mismatch: %+v", found)
	}

	// Test Delete
	if err := db.Delete(addedID); err != nil {
		t.Fatalf("Failed to delete device: %v", err)
	}

	devices, _, _ = db.GetAll()
	if len(devices) != 2 {
		t.Fatalf("Expected 2 devices after delete, got %d", len(devices))
	}

	// Verify backup file was created
	bakContent, err := os.ReadFile(tmpDB + ".bak")
	if err != nil {
		t.Fatalf("Expected backup file to exist: %v", err)
	}
	if !strings.Contains(string(bakContent), "sw1") {
		t.Errorf("Expected backup to contain sw1")
	}
}
