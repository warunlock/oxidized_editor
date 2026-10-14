package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"oxidized_edit/internal/auth"
	"oxidized_edit/internal/config"
	"oxidized_edit/internal/database"
	"oxidized_edit/internal/service"
)

type API struct {
	cfg          *config.Config
	routerDB     *database.RouterDB
	oxidized     *service.OxidizedManager
	sessionStore *auth.SessionStore
}

func NewAPI(cfg *config.Config, db *database.RouterDB, svc *service.OxidizedManager, sessions *auth.SessionStore) *API {
	return &API{
		cfg:          cfg,
		routerDB:     db,
		oxidized:     svc,
		sessionStore: sessions,
	}
}

// JSON helper
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// AuthMiddleware protects routes requiring login
func (a *API) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := auth.GetTokenFromRequest(r)
		username, valid := a.sessionStore.ValidateToken(token)
		if !valid || username == "" {
			writeError(w, http.StatusUnauthorized, "Authentication required. Please log in.")
			return
		}
		next(w, r)
	}
}

// LoginHandler handles user authentication
func (a *API) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username != a.cfg.GetUsername() || !a.cfg.VerifyPassword(req.Password) {
		time.Sleep(500 * time.Millisecond) // Thwart brute force timing
		writeError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}

	token := a.sessionStore.CreateSession(req.Username)
	auth.SetSessionCookie(w, token, 86400) // 24 hours

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":  true,
		"username": req.Username,
		"token":    token,
	})
}

// LogoutHandler terminates the user session
func (a *API) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	token := auth.GetTokenFromRequest(r)
	if token != "" {
		a.sessionStore.DestroySession(token)
	}
	auth.ClearSessionCookie(w)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	})
}

// MeHandler checks authentication status and returns basic info
func (a *API) MeHandler(w http.ResponseWriter, r *http.Request) {
	token := auth.GetTokenFromRequest(r)
	username, valid := a.sessionStore.ValidateToken(token)

	if !valid {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"username":      username,
		"db_path":       a.cfg.GetDBPath(),
		"port":          a.cfg.GetPort(),
	})
}

// ChangePasswordHandler updates the password in config.yaml
func (a *API) ChangePasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if !a.cfg.VerifyPassword(req.CurrentPassword) {
		writeError(w, http.StatusBadRequest, "Current password is incorrect")
		return
	}

	if len(req.NewPassword) < 4 {
		writeError(w, http.StatusBadRequest, "New password must be at least 4 characters long")
		return
	}

	if err := a.cfg.UpdatePassword(req.NewPassword); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to update password: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Password changed successfully and updated in config file",
	})
}

// GetDevicesHandler returns the list of all devices
func (a *API) GetDevicesHandler(w http.ResponseWriter, r *http.Request) {
	devices, modTime, dbPath := a.routerDB.GetAll()

	// Extract unique models and groups for filters/stats
	modelsMap := make(map[string]int)
	groupsMap := make(map[string]int)
	for _, d := range devices {
		if d.Model != "" {
			modelsMap[d.Model]++
		}
		if d.Group != "" {
			groupsMap[d.Group]++
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success":       true,
		"devices":       devices,
		"count":         len(devices),
		"models":        modelsMap,
		"groups":        groupsMap,
		"db_path":       dbPath,
		"last_modified": modTime.Format(time.RFC3339),
	})
}

// AddDeviceHandler adds a new device record
func (a *API) AddDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var dev database.Device
	if err := json.NewDecoder(r.Body).Decode(&dev); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid device JSON")
		return
	}

	if err := a.routerDB.Add(&dev); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Device added successfully",
		"device":  dev,
	})
}

// UpdateDeviceHandler updates an existing device
func (a *API) UpdateDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Extract ID from path: /api/devices/{id}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		writeError(w, http.StatusBadRequest, "Device ID missing in URL path")
		return
	}
	id := pathParts[2]

	var dev database.Device
	if err := json.NewDecoder(r.Body).Decode(&dev); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid device JSON")
		return
	}

	if err := a.routerDB.Update(id, dev); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Device updated successfully",
	})
}

// DeleteDeviceHandler deletes a device by ID
func (a *API) DeleteDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 {
		writeError(w, http.StatusBadRequest, "Device ID missing in URL path")
		return
	}
	id := pathParts[2]

	if err := a.routerDB.Delete(id); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Device deleted successfully",
	})
}

// ReloadDBHandler reloads router.db from disk
func (a *API) ReloadDBHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.routerDB.Load(); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to reload router.db: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Database reloaded from disk",
	})
}

// GetRawHandler returns raw router.db content
func (a *API) GetRawHandler(w http.ResponseWriter, r *http.Request) {
	content, err := a.routerDB.GetRaw()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to read raw file: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"content": content,
		"path":    a.cfg.GetDBPath(),
	})
}

// SaveRawHandler overwrites raw router.db content
func (a *API) SaveRawHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := a.routerDB.SaveRaw(req.Content); err != nil {
		writeError(w, http.StatusBadRequest, "Failed to save raw file: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Raw file saved successfully",
	})
}

// RestartOxidizedHandler restarts the oxidized service
func (a *API) RestartOxidizedHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	res := a.oxidized.Restart(r.Context())
	writeJSON(w, http.StatusOK, res)
}

// OxidizedStatusHandler returns the status of oxidized service
func (a *API) OxidizedStatusHandler(w http.ResponseWriter, r *http.Request) {
	res := a.oxidized.Status(r.Context())
	writeJSON(w, http.StatusOK, res)
}
