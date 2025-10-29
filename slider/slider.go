package slider

import (
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Slider struct {
	ID          int       `json:"slider_id"`
	ImageLink   string    `json:"banner_image_url"`
	ForwardLink string    `json:"redirect_url"`
	Title       string    `json:"banner_title"`
	Order       int       `json:"display_order"`
	IsActive    bool      `json:"is_visible"`
	CreatedAt   time.Time `json:"created_date"`
}

var db *sql.DB

// InitDB initializes the database connection
func InitDB(database *sql.DB) {
	db = database
	createTable()
}

// Create sliders table and uploads directory
func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS sliders (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		image_link TEXT NOT NULL,
		forward_link TEXT,
		title TEXT,
		order_num INTEGER DEFAULT 0,
		is_active INTEGER DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_slider_active ON sliders(is_active);
	CREATE INDEX IF NOT EXISTS idx_slider_order ON sliders(order_num);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("❌ Error creating sliders table: %v", err)
	} else {
		log.Println("✅ Sliders table ready")
	}
}

// GetActiveSliders retrieves all active sliders ordered by order_num
func GetActiveSliders() ([]Slider, error) {
	query := `
		SELECT id, image_link, forward_link, title, order_num, is_active, created_at
		FROM sliders
		WHERE is_active = 1
		ORDER BY order_num ASC, created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sliders []Slider
	for rows.Next() {
		var slider Slider
		err := rows.Scan(&slider.ID, &slider.ImageLink, &slider.ForwardLink,
			&slider.Title, &slider.Order, &slider.IsActive, &slider.CreatedAt)
		if err != nil {
			log.Printf("Error scanning slider: %v", err)
			continue
		}
		sliders = append(sliders, slider)
	}

	return sliders, nil
}

// GetAllSlidersForAdmin retrieves all sliders (including inactive)
func GetAllSlidersForAdmin() ([]Slider, error) {
	query := `
		SELECT id, image_link, forward_link, title, order_num, is_active, created_at
		FROM sliders
		ORDER BY order_num ASC, created_at DESC
	`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sliders []Slider
	for rows.Next() {
		var slider Slider
		err := rows.Scan(&slider.ID, &slider.ImageLink, &slider.ForwardLink,
			&slider.Title, &slider.Order, &slider.IsActive, &slider.CreatedAt)
		if err != nil {
			log.Printf("Error scanning slider: %v", err)
			continue
		}
		sliders = append(sliders, slider)
	}

	return sliders, nil
}

// InsertSlider adds a new slider
func InsertSlider(slider Slider) error {
	query := `
		INSERT INTO sliders (image_link, forward_link, title, order_num, is_active)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.Exec(query, slider.ImageLink, slider.ForwardLink,
		slider.Title, slider.Order, slider.IsActive)
	if err != nil {
		log.Printf("❌ Error inserting slider: %v", err)
		return err
	}
	log.Printf("✅ Slider inserted: %s", slider.Title)
	return nil
}

// UpdateSlider updates an existing slider
func UpdateSlider(slider Slider) error {
	query := `
		UPDATE sliders
		SET image_link = $1, forward_link = $2, title = $3, order_num = $4, is_active = $5
		WHERE id = $6
	`
	_, err := db.Exec(query, slider.ImageLink, slider.ForwardLink,
		slider.Title, slider.Order, slider.IsActive, slider.ID)
	if err != nil {
		log.Printf("❌ Error updating slider: %v", err)
		return err
	}
	log.Printf("✅ Slider updated: %s", slider.Title)
	return nil
}

// DeleteSlider deletes a slider
func DeleteSlider(id int) error {
	query := `DELETE FROM sliders WHERE id = $1`
	_, err := db.Exec(query, id)
	if err != nil {
		log.Printf("❌ Error deleting slider: %v", err)
		return err
	}
	log.Printf("✅ Slider deleted: ID %d", id)
	return nil
}

// GetSlidersHandler returns active sliders
func GetSlidersHandler(c *gin.Context) {
	sliders, err := GetActiveSliders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sliders)
}
