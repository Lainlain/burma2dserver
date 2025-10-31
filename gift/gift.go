package gift

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"burma2d/fcm"

	"github.com/gin-gonic/gin"
)

type Gift struct {
	ID          int       `json:"gift_id"`
	Name        string    `json:"gift_name"`
	ImageLink   string    `json:"image_url"`
	Type        string    `json:"reward_type"`
	Description string    `json:"gift_description"`
	Points      int       `json:"required_points"`
	Stock       int       `json:"available_stock"`
	IsActive    bool      `json:"is_available"`
	CreatedAt   time.Time `json:"created_date"`
}

var db *sql.DB

// InitDB initializes the database connection
func InitDB(database *sql.DB) {
	db = database
	createTable()
}

type GiftType struct {
	ID        int       `json:"type_id"`
	Name      string    `json:"type_name"`
	CreatedAt time.Time `json:"created_date"`
}

// Create gifts table and gift_types table
func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS gift_types (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS gifts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		image_link TEXT NOT NULL,
		type TEXT NOT NULL,
		description TEXT,
		points INTEGER DEFAULT 0,
		stock INTEGER DEFAULT 0,
		is_active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_gift_type ON gifts(type);
	CREATE INDEX IF NOT EXISTS idx_gift_active ON gifts(is_active);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("❌ Error creating gifts tables: %v", err)
	} else {
		log.Println("✅ Gifts tables ready")
	}
}

// GetAllGifts retrieves all active gifts grouped by type
func GetAllGifts() (map[string][]Gift, error) {
	query := `
		SELECT id, name, image_link, type, description, points, stock, is_active, created_at
		FROM gifts
		WHERE is_active = true
		ORDER BY type, created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	giftsMap := make(map[string][]Gift)
	for rows.Next() {
		var gift Gift
		err := rows.Scan(&gift.ID, &gift.Name, &gift.ImageLink, &gift.Type,
			&gift.Description, &gift.Points, &gift.Stock, &gift.IsActive, &gift.CreatedAt)
		if err != nil {
			log.Printf("Error scanning gift: %v", err)
			continue
		}
		giftsMap[gift.Type] = append(giftsMap[gift.Type], gift)
	}

	return giftsMap, nil
}

// GetAllGiftsForAdmin retrieves all gifts (including inactive)
func GetAllGiftsForAdmin() ([]Gift, error) {
	query := `
		SELECT id, name, image_link, type, description, points, stock, is_active, created_at
		FROM gifts
		ORDER BY created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gifts []Gift
	for rows.Next() {
		var gift Gift
		err := rows.Scan(&gift.ID, &gift.Name, &gift.ImageLink, &gift.Type,
			&gift.Description, &gift.Points, &gift.Stock, &gift.IsActive, &gift.CreatedAt)
		if err != nil {
			log.Printf("Error scanning gift: %v", err)
			continue
		}
		gifts = append(gifts, gift)
	}

	return gifts, nil
}

// InsertGift adds a new gift
func InsertGift(gift Gift) error {
	query := `
		INSERT INTO gifts (name, image_link, type, description, points, stock, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := db.Exec(query, gift.Name, gift.ImageLink, gift.Type,
		gift.Description, gift.Points, gift.Stock, gift.IsActive)
	if err != nil {
		log.Printf("❌ Error inserting gift: %v", err)
		return err
	}
	log.Printf("✅ Gift inserted: %s", gift.Name)
	return nil
}

// UpdateGift updates an existing gift
func UpdateGift(gift Gift) error {
	query := `
		UPDATE gifts
		SET name = $1, image_link = $2, type = $3, description = $4,
		    points = $5, stock = $6, is_active = $7
		WHERE id = $8
	`
	_, err := db.Exec(query, gift.Name, gift.ImageLink, gift.Type,
		gift.Description, gift.Points, gift.Stock, gift.IsActive, gift.ID)
	if err != nil {
		log.Printf("❌ Error updating gift: %v", err)
		return err
	}
	log.Printf("✅ Gift updated: %s", gift.Name)

	// Send FCM notification about gift availability
	go func() {
		if err := fcm.SendGiftAvailableNotification(gift.Name); err != nil {
			log.Printf("⚠️ Failed to send FCM notification for gift '%s': %v", gift.Name, err)
		}
	}()

	return nil
}

// DeleteGift deletes a gift
func DeleteGift(id int) error {
	query := `DELETE FROM gifts WHERE id = $1`
	_, err := db.Exec(query, id)
	if err != nil {
		log.Printf("❌ Error deleting gift: %v", err)
		return err
	}
	log.Printf("✅ Gift deleted: ID %d", id)
	return nil
}

// Gift Type CRUD Operations
func GetAllGiftTypes() ([]GiftType, error) {
	query := `SELECT id, name, created_at FROM gift_types ORDER BY name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []GiftType
	for rows.Next() {
		var giftType GiftType
		if err := rows.Scan(&giftType.ID, &giftType.Name, &giftType.CreatedAt); err != nil {
			continue
		}
		types = append(types, giftType)
	}

	return types, nil
}

func InsertGiftType(name string) error {
	query := `INSERT INTO gift_types (name) VALUES ($1)`
	_, err := db.Exec(query, name)
	if err != nil {
		log.Printf("❌ Error inserting gift type: %v", err)
		return err
	}
	log.Printf("✅ Gift type inserted: %s", name)
	return nil
}

func UpdateGiftType(id int, name string) error {
	query := `UPDATE gift_types SET name = $1 WHERE id = $2`
	_, err := db.Exec(query, name, id)
	if err != nil {
		log.Printf("❌ Error updating gift type: %v", err)
		return err
	}
	log.Printf("✅ Gift type updated: %s", name)
	return nil
}

func DeleteGiftType(id int) error {
	query := `DELETE FROM gift_types WHERE id = $1`
	_, err := db.Exec(query, id)
	if err != nil {
		log.Printf("❌ Error deleting gift type: %v", err)
		return err
	}
	log.Printf("✅ Gift type deleted: ID %d", id)
	return nil
}

// GetGiftTypes returns distinct gift types (backwards compatibility)
func GetGiftTypes() ([]string, error) {
	query := `SELECT DISTINCT type FROM gifts ORDER BY type`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var types []string
	for rows.Next() {
		var giftType string
		if err := rows.Scan(&giftType); err != nil {
			continue
		}
		types = append(types, giftType)
	}

	return types, nil
}

// GetGiftTypesHandler returns distinct gift types
func GetGiftTypesHandler(c *gin.Context) {
	types, err := GetGiftTypes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, types)
}

// GetAllGiftTypesHandler returns all gift types from gift_types table
func GetAllGiftTypesHandler(c *gin.Context) {
	types, err := GetAllGiftTypes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, types)
}

// CreateGiftTypeHandler creates a new gift type
func CreateGiftTypeHandler(c *gin.Context) {
	var req struct {
		TypeName string `json:"type_name"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := InsertGiftType(req.TypeName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gift type created successfully"})
}

// UpdateGiftTypeHandler updates a gift type
func UpdateGiftTypeHandler(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		TypeName string `json:"type_name"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	typeID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := UpdateGiftType(typeID, req.TypeName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gift type updated successfully"})
}

// DeleteGiftTypeHandler deletes a gift type
func DeleteGiftTypeHandler(c *gin.Context) {
	id := c.Param("id")
	typeID, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := DeleteGiftType(typeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Gift type deleted successfully"})
}

// GetGiftsHandler returns gifts grouped by type
func GetGiftsHandler(c *gin.Context) {
	gifts, err := GetAllGifts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gifts)
}
