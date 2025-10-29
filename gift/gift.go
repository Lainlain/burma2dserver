package gift

import (
	"database/sql"
	"encoding/json"
	"log"
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

// Create gifts table and uploads directory
func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS gifts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		image_link TEXT NOT NULL,
		type TEXT NOT NULL CHECK (type IN ('Daily', 'Weekly')),
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
		log.Printf("❌ Error creating gifts table: %v", err)
	} else {
		log.Println("✅ Gifts table ready")
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

// GetGiftsHandler returns gifts grouped by type
func GetGiftsHandler(c *gin.Context) {
	gifts, err := GetAllGifts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gifts)
}
