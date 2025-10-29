package appconfig

import (
"database/sql"
"fmt"
"net/http"

"github.com/gin-gonic/gin"
)

type AppConfig struct {
ID                 int    `json:"id"`
LatestVersion      string `json:"latest_version"`
MinimumVersion     string `json:"minimum_version"`
UpdateRequired     bool   `json:"update_required"`
UpdateURL          string `json:"update_url"`
UpdateMessage      string `json:"update_message"`
MaintenanceMode    bool   `json:"maintenance_mode"`
MaintenanceMessage string `json:"maintenance_message"`
ForceUpdate        bool   `json:"force_update"`
AppEnabled         bool   `json:"app_enabled"`
CreatedAt          string `json:"created_at"`
UpdatedAt          string `json:"updated_at"`
}

type VersionCheckRequest struct {
Version string `form:"version" binding:"required"`
}

type VersionCheckResponse struct {
UpdateRequired     bool   `json:"update_required"`
UpdateURL          string `json:"update_url,omitempty"`
UpdateMessage      string `json:"update_message,omitempty"`
MaintenanceMode    bool   `json:"maintenance_mode"`
MaintenanceMessage string `json:"maintenance_message,omitempty"`
ForceUpdate        bool   `json:"force_update"`
AppEnabled         bool   `json:"app_enabled"`
}

var db *sql.DB

func InitDB(database *sql.DB) error {
db = database

createTableSQL := `CREATE TABLE IF NOT EXISTS app_config (
INTEGER PRIMARY KEY AUTOINCREMENT,
 TEXT NOT NULL,
imum_version TEXT NOT NULL,
uired BOOLEAN DEFAULT 0,
TEXT,
TEXT,
tenance_mode BOOLEAN DEFAULT 0,
tenance_message TEXT,
BOOLEAN DEFAULT 0,
abled BOOLEAN DEFAULT 1,
DATETIME DEFAULT CURRENT_TIMESTAMP,
DATETIME DEFAULT CURRENT_TIMESTAMP
)`

_, err := db.Exec(createTableSQL)
if err != nil {
 fmt.Errorf("failed to create app_config table: %v", err)
}

fmt.Println("✅ App config table created/verified")
insertDefaultConfig()

return nil
}

func insertDefaultConfig() {
var count int
err := db.QueryRow("SELECT COUNT(*) FROM app_config").Scan(&count)
if err != nil {
tf("Error checking app_config: %v\n", err)

}

if count == 0 {
uery := `INSERT INTO app_config (latest_version, minimum_version, update_url, update_message, maintenance_message) VALUES (?, ?, ?, ?, ?)`
err = db.Exec(query, "1.0.0", "1.0.0", "https://play.google.com/store/apps/details?id=com.twod.expect", "New version available", "App is under maintenance")
err != nil {
tf("❌ Failed to insert default config: %v\n", err)
else {
tln("✅ Default app config inserted successfully")
else {
tf("ℹ️  App config already exists (%d records)\n", count)
}
}

func GetAppConfig(c *gin.Context) {
var config AppConfig
query := `SELECT id, latest_version, minimum_version, update_required, update_url, update_message, maintenance_mode, maintenance_message, force_update, app_enabled, created_at, updated_at FROM app_config ORDER BY id DESC LIMIT 1`

err := db.QueryRow(query).Scan(&config.ID, &config.LatestVersion, &config.MinimumVersion, &config.UpdateRequired, &config.UpdateURL, &config.UpdateMessage, &config.MaintenanceMode, &config.MaintenanceMessage, &config.ForceUpdate, &config.AppEnabled, &config.CreatedAt, &config.UpdatedAt)
if err != nil {
(http.StatusInternalServerError, gin.H{"error": "Failed to fetch app config"})

}

c.JSON(http.StatusOK, config)
}

func CheckVersion(c *gin.Context) {
var req VersionCheckRequest
if err := c.ShouldBindQuery(&req); err != nil {
(http.StatusBadRequest, gin.H{"error": "Version parameter is required"})

}

var config AppConfig
query := `SELECT latest_version, minimum_version, update_url, update_message, maintenance_mode, maintenance_message, force_update, app_enabled FROM app_config ORDER BY id DESC LIMIT 1`

err := db.QueryRow(query).Scan(&config.LatestVersion, &config.MinimumVersion, &config.UpdateURL, &config.UpdateMessage, &config.MaintenanceMode, &config.MaintenanceMessage, &config.ForceUpdate, &config.AppEnabled)
if err != nil {
(http.StatusInternalServerError, gin.H{"error": "Failed to fetch app config"})

}

response := VersionCheckResponse{
uired:     compareVersions(req.Version, config.MinimumVersion),
         config.UpdateURL,
     config.UpdateMessage,
tenanceMode:    config.MaintenanceMode,
tenanceMessage: config.MaintenanceMessage,
       config.ForceUpdate,
abled:         config.AppEnabled,
}

c.JSON(http.StatusOK, response)
}

func UpdateAppConfig(c *gin.Context) {
var config AppConfig
if err := c.ShouldBindJSON(&config); err != nil {
(http.StatusBadRequest, gin.H{"error": err.Error()})

}

query := `UPDATE app_config SET latest_version=?, minimum_version=?, update_required=?, update_url=?, update_message=?, maintenance_mode=?, maintenance_message=?, force_update=?, app_enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`

_, err := db.Exec(query, config.LatestVersion, config.MinimumVersion, config.UpdateRequired, config.UpdateURL, config.UpdateMessage, config.MaintenanceMode, config.MaintenanceMessage, config.ForceUpdate, config.AppEnabled, config.ID)
if err != nil {
(http.StatusInternalServerError, gin.H{"error": "Failed to update app config"})

}

c.JSON(http.StatusOK, gin.H{"message": "App config updated successfully"})
}

func compareVersions(currentVersion, minimumVersion string) bool {
return currentVersion < minimumVersion
}
