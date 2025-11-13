package paper

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

var db *sql.DB

type PaperType struct {
	ID           int       `json:"type_id"`
	Name         string    `json:"type_name"`
	DisplayOrder int       `json:"display_order"`
	IsActive     bool      `json:"is_active"`
	ImageCount   int       `json:"image_count"`
	CreatedAt    time.Time `json:"created_date"`
	UpdatedAt    time.Time `json:"modified_date"`
}

type PaperImage struct {
	ID           int       `json:"image_id"`
	TypeID       int       `json:"type_id"`
	TypeName     string    `json:"type_name,omitempty"`
	ImageURL     string    `json:"image_url"`
	DisplayOrder int       `json:"display_order"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_date"`
	UpdatedAt    time.Time `json:"modified_date"`
}

type PaperTypeWithImages struct {
	Type   PaperType    `json:"type"`
	Images []PaperImage `json:"images"`
}

func InitDB(database *sql.DB) {
	db = database
	createTables()
}

func createTables() {
	// Enable foreign keys
	_, err := db.Exec(`PRAGMA foreign_keys = ON`)
	if err != nil {
		panic("Failed to enable foreign keys: " + err.Error())
	}

	// Create paper_types table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS paper_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			display_order INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		panic("Failed to create paper_types table: " + err.Error())
	}

	// Create paper_images table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS paper_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type_id INTEGER NOT NULL REFERENCES paper_types(id) ON DELETE CASCADE,
			image_url TEXT NOT NULL,
			display_order INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		panic("Failed to create paper_images table: " + err.Error())
	}

	// Create indexes
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_paper_images_type_id ON paper_images(type_id)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_paper_images_display_order ON paper_images(display_order)`)
	db.Exec(`CREATE INDEX IF NOT EXISTS idx_paper_types_display_order ON paper_types(display_order)`)

	// Insert sample data if table is empty
	var count int
	db.QueryRow("SELECT COUNT(*) FROM paper_types").Scan(&count)
	if count == 0 {
		db.Exec(`
			INSERT INTO paper_types (name, display_order) VALUES
			('Myanmar News', 1),
			('Thailand News', 2),
			('International', 3)
		`)
	}
}

// Get all paper types with image count
func GetAllTypes(c *gin.Context) {
	rows, err := db.Query(`
		SELECT pt.id, pt.name, pt.display_order, pt.is_active, pt.created_at, pt.updated_at,
		       COALESCE(COUNT(pi.id), 0) as image_count
		FROM paper_types pt
		LEFT JOIN paper_images pi ON pt.id = pi.type_id AND pi.is_active = 1
		WHERE pt.is_active = 1
		GROUP BY pt.id, pt.name, pt.display_order, pt.is_active, pt.created_at, pt.updated_at
		ORDER BY pt.display_order ASC, pt.name ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var types []PaperType
	for rows.Next() {
		var t PaperType
		if err := rows.Scan(&t.ID, &t.Name, &t.DisplayOrder, &t.IsActive, &t.CreatedAt, &t.UpdatedAt, &t.ImageCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		types = append(types, t)
	}

	c.JSON(http.StatusOK, types)
}

// Get all paper types with their images (for admin)
func GetAllTypesWithImages(c *gin.Context) {
	// Get all types
	typeRows, err := db.Query(`
		SELECT id, name, display_order, is_active, created_at, updated_at
		FROM paper_types
		ORDER BY display_order ASC, name ASC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer typeRows.Close()

	var results []PaperTypeWithImages
	for typeRows.Next() {
		var t PaperType
		if err := typeRows.Scan(&t.ID, &t.Name, &t.DisplayOrder, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Get images for this type
		imageRows, err := db.Query(`
			SELECT id, type_id, image_url, display_order, is_active, created_at, updated_at
			FROM paper_images
			WHERE type_id = ?
			ORDER BY display_order ASC, created_at DESC
		`, t.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var images []PaperImage
		for imageRows.Next() {
			var img PaperImage
			if err := imageRows.Scan(&img.ID, &img.TypeID, &img.ImageURL, &img.DisplayOrder, &img.IsActive, &img.CreatedAt, &img.UpdatedAt); err != nil {
				imageRows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			images = append(images, img)
		}
		imageRows.Close()

		results = append(results, PaperTypeWithImages{
			Type:   t,
			Images: images,
		})
	}

	c.JSON(http.StatusOK, results)
}

// Get images by type ID
func GetImagesByType(c *gin.Context) {
	typeID := c.Param("type_id")

	rows, err := db.Query(`
		SELECT pi.id, pi.type_id, pt.name, pi.image_url, pi.display_order, pi.is_active, pi.created_at, pi.updated_at
		FROM paper_images pi
		JOIN paper_types pt ON pi.type_id = pt.id
		WHERE pi.type_id = ? AND pi.is_active = 1 AND pt.is_active = 1
		ORDER BY pi.display_order ASC, pi.created_at DESC
	`, typeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var images []PaperImage
	for rows.Next() {
		var img PaperImage
		if err := rows.Scan(&img.ID, &img.TypeID, &img.TypeName, &img.ImageURL, &img.DisplayOrder, &img.IsActive, &img.CreatedAt, &img.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		images = append(images, img)
	}

	c.JSON(http.StatusOK, images)
}

// Create paper type
func CreateType(c *gin.Context) {
	var input struct {
		Name         string `json:"name" binding:"required"`
		DisplayOrder int    `json:"display_order"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec(`
		INSERT INTO paper_types (name, display_order, is_active, created_at, updated_at)
		VALUES (?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, input.Name, input.DisplayOrder)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Paper type created successfully"})
}

// Update paper type
func UpdateType(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Name         string `json:"name"`
		DisplayOrder int    `json:"display_order"`
		IsActive     bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`
		UPDATE paper_types
		SET name = ?, display_order = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, input.Name, input.DisplayOrder, input.IsActive, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Paper type updated successfully"})
}

// Delete paper type
func DeleteType(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec("DELETE FROM paper_types WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Paper type deleted successfully"})
}

// Create paper image
func CreateImage(c *gin.Context) {
	var input struct {
		TypeID       int    `json:"type_id" binding:"required"`
		ImageURL     string `json:"image_url" binding:"required"`
		DisplayOrder int    `json:"display_order"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec(`
		INSERT INTO paper_images (type_id, image_url, display_order, is_active, created_at, updated_at)
		VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, input.TypeID, input.ImageURL, input.DisplayOrder)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Paper image created successfully"})
}

// Update paper image
func UpdateImage(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		TypeID       int    `json:"type_id"`
		ImageURL     string `json:"image_url"`
		DisplayOrder int    `json:"display_order"`
		IsActive     bool   `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`
		UPDATE paper_images
		SET type_id = ?, image_url = ?, display_order = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, input.TypeID, input.ImageURL, input.DisplayOrder, input.IsActive, id)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Paper image updated successfully"})
}

// Delete paper image
func DeleteImage(c *gin.Context) {
	id := c.Param("id")

	// First, get the image URL to delete the actual file
	var imageURL string
	err := db.QueryRow("SELECT image_url FROM paper_images WHERE id = ?", id).Scan(&imageURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Image not found"})
		return
	}

	// Delete from database first
	_, err = db.Exec("DELETE FROM paper_images WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Extract filename from URL and delete the physical file
	// URL format: http://localhost:4545/api/images/filename.png
	// We need to extract the filename part
	if imageURL != "" {
		// Split by "/" and get the last part (filename)
		parts := splitURL(imageURL)
		if len(parts) > 0 {
			filename := parts[len(parts)-1]
			deletePhysicalFile(filename)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Paper image deleted successfully"})
}

// Helper function to split URL
func splitURL(url string) []string {
	var parts []string
	currentPart := ""
	for _, char := range url {
		if char == '/' {
			if currentPart != "" {
				parts = append(parts, currentPart)
				currentPart = ""
			}
		} else {
			currentPart += string(char)
		}
	}
	if currentPart != "" {
		parts = append(parts, currentPart)
	}
	return parts
}

// Helper function to delete physical file
func deletePhysicalFile(filename string) {
	// Import required packages at top of file: "os" and "path/filepath"
	filepath := "uploads/" + filename
	os.Remove(filepath) // Ignore errors - file might already be deleted
}

// Batch create images
func BatchCreateImages(c *gin.Context) {
	var input struct {
		TypeID    int      `json:"type_id" binding:"required"`
		ImageURLs []string `json:"image_urls" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var insertedIDs []int
	for i, url := range input.ImageURLs {
		result, err := tx.Exec(`
			INSERT INTO paper_images (type_id, image_url, display_order, is_active, created_at, updated_at)
			VALUES (?, ?, ?, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, input.TypeID, url, i)

		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		id, err := result.LastInsertId()
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		insertedIDs = append(insertedIDs, int(id))
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Images created successfully",
		"count":   len(insertedIDs),
		"ids":     insertedIDs,
	})
}

// Get type by ID
func GetTypeByID(id string) (*PaperType, error) {
	var t PaperType
	err := db.QueryRow(`
		SELECT id, name, display_order, is_active, created_at, updated_at
		FROM paper_types
		WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.DisplayOrder, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Get image count
	db.QueryRow(`
		SELECT COUNT(*) FROM paper_images WHERE type_id = ? AND is_active = 1
	`, id).Scan(&t.ImageCount)

	return &t, nil
}

// Get image by ID
func GetImageByID(id string) (*PaperImage, error) {
	var img PaperImage
	err := db.QueryRow(`
		SELECT pi.id, pi.type_id, pt.name, pi.image_url, pi.display_order, pi.is_active, pi.created_at, pi.updated_at
		FROM paper_images pi
		JOIN paper_types pt ON pi.type_id = pt.id
		WHERE pi.id = ?
	`, id).Scan(&img.ID, &img.TypeID, &img.TypeName, &img.ImageURL, &img.DisplayOrder, &img.IsActive, &img.CreatedAt, &img.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &img, nil
}

// Get next display order for type
func GetNextDisplayOrder(typeID int) int {
	var order int
	err := db.QueryRow(`
		SELECT COALESCE(MAX(display_order), 0) + 1
		FROM paper_images
		WHERE type_id = ?
	`, typeID).Scan(&order)

	if err != nil {
		return 1
	}
	return order
}

// Delete all images by type ID
func DeleteAllImagesByType(c *gin.Context) {
	typeID := c.Param("type_id")

	// First, get all image URLs to delete physical files
	rows, err := db.Query("SELECT image_url FROM paper_images WHERE type_id = ?", typeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var imageURLs []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		imageURLs = append(imageURLs, url)
	}

	// Delete all images from database
	result, err := db.Exec("DELETE FROM paper_images WHERE type_id = ?", typeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	// Delete physical files
	for _, imageURL := range imageURLs {
		if imageURL != "" {
			parts := splitURL(imageURL)
			if len(parts) > 0 {
				filename := parts[len(parts)-1]
				deletePhysicalFile(filename)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "All images deleted successfully",
		"count":   rowsAffected,
	})
}
