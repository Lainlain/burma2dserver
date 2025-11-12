package admin

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var db *sql.DB

// InitDB initializes the database connection for admin
func InitDB(database *sql.DB) {
	db = database
}

// AdminDashboardHandler renders the admin dashboard home
func AdminDashboardHandler(c *gin.Context) {
	c.HTML(200, "dashboard.html", gin.H{
		"title": "Admin Dashboard - Burma 2D 2025",
	})
}

// ManageGiftsPageHandler renders the gifts management page
func ManageGiftsPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "manage_gifts.html", gin.H{
		"title": "Manage Gifts - Admin",
	})
}

// ManageSlidersPageHandler renders the sliders management page
func ManageSlidersPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "manage_sliders.html", gin.H{
		"title": "Manage Sliders - Admin",
	})
}

// CreateGiftPageHandler renders the create gift form
func CreateGiftPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "create_gift.html", gin.H{
		"title": "Create Gift - Admin",
	})
}

// CreateSliderPageHandler renders the create slider form
func CreateSliderPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "create_slider.html", gin.H{
		"title": "Create Slider - Admin",
	})
}

// EditGiftPageHandler renders the edit gift form
func EditGiftPageHandler(c *gin.Context) {
	id := c.Param("id")
	c.HTML(http.StatusOK, "edit_gift.html", gin.H{
		"title": "Edit Gift - Admin",
		"id":    id,
	})
}

// EditSliderPageHandler renders the edit slider form
func EditSliderPageHandler(c *gin.Context) {
	id := c.Param("id")
	c.HTML(http.StatusOK, "edit_slider.html", gin.H{
		"title": "Edit Slider - Admin",
		"id":    id,
	})
}

// GetGiftByIDHandler returns a single gift by ID
func GetGiftByIDHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	query := `
		SELECT id, name, image_link, type, description, points, stock, is_active, created_at
		FROM gifts WHERE id = $1
	`
	var gift struct {
		ID          int    `json:"id"`
		Name        string `json:"name"`
		ImageLink   string `json:"image_link"`
		Type        string `json:"type"`
		Description string `json:"description"`
		Points      int    `json:"points"`
		Stock       int    `json:"stock"`
		IsActive    bool   `json:"is_active"`
		CreatedAt   string `json:"created_at"`
	}

	err = db.QueryRow(query, id).Scan(&gift.ID, &gift.Name, &gift.ImageLink,
		&gift.Type, &gift.Description, &gift.Points, &gift.Stock, &gift.IsActive, &gift.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gift not found"})
		return
	}

	c.JSON(http.StatusOK, gift)
}

// GetSliderByIDHandler returns a single slider by ID
func GetSliderByIDHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	query := `
		SELECT id, image_link, forward_link, title, order_num, is_active, created_at
		FROM sliders WHERE id = $1
	`
	var slider struct {
		ID          int    `json:"id"`
		ImageLink   string `json:"image_link"`
		ForwardLink string `json:"forward_link"`
		Title       string `json:"title"`
		Order       int    `json:"order"`
		IsActive    bool   `json:"is_active"`
		CreatedAt   string `json:"created_at"`
	}

	err = db.QueryRow(query, id).Scan(&slider.ID, &slider.ImageLink, &slider.ForwardLink,
		&slider.Title, &slider.Order, &slider.IsActive, &slider.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Slider not found"})
		return
	}

	c.JSON(http.StatusOK, slider)
}

// UploadImageHandler handles image uploads and returns the file path
// Supports both local storage and Cloudflare R2 (controlled by USE_R2 env var)
func UploadImageHandler(c *gin.Context) {
	// Get the file from form data
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}

	// Validate file type
	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Only jpg, png, gif, webp allowed"})
		return
	}

	// If R2 is enabled, upload to Cloudflare R2
	if IsR2Enabled() {
		imageURL, err := UploadToR2(file)
		if err != nil {
			log.Printf("❌ R2 upload failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload to R2"})
			return
		}

		log.Printf("✅ R2 upload successful: %s", imageURL)
		c.JSON(http.StatusOK, gin.H{
			"success":   true,
			"image_url": imageURL,
			"filename":  filepath.Base(imageURL),
			"storage":   "r2",
		})
		return
	}

	// Otherwise, use local storage (original behavior)
	log.Println("📁 Using local storage (R2 disabled)")

	// Get uploads directory from env or use default
	uploadsDir := os.Getenv("UPLOADS_PATH")
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}

	// Create uploads directory if not exists with 755 permissions
	if err := os.MkdirAll(uploadsDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create uploads directory"})
		return
	}

	// FORCE uploads directory to 755 - this is critical for nginx/cloudflare access
	// Using multiple methods to ensure it sticks
	os.Chmod(uploadsDir, 0755)

	// Verify permissions were set
	if info, err := os.Stat(uploadsDir); err == nil {
		log.Printf("📁 Uploads dir permissions after chmod: %s", info.Mode().Perm())
	}

	// Generate unique filename using timestamp
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("%d_%s", timestamp, filepath.Base(file.Filename))
	filePath := filepath.Join(uploadsDir, filename)

	// Save the file
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save image"})
		return
	}

	// Set file permissions to 644 (readable by everyone)
	os.Chmod(filePath, 0644)

	// FORCE directory permissions again after file save
	os.Chmod(uploadsDir, 0755)

	log.Printf("💾 Image saved: %s (path: %s, file perms: 644, dir perms: 755)", filename, filePath)

	// Build URL dynamically based on the incoming request
	// Detect HTTPS from multiple sources (direct TLS, proxy headers, or port)
	scheme := "http"

	// Log all relevant headers for debugging
	log.Printf("🔍 Image Upload - Host: %s, TLS: %v", c.Request.Host, c.Request.TLS != nil)
	log.Printf("🔍 X-Forwarded-Proto: %s", c.GetHeader("X-Forwarded-Proto"))
	log.Printf("🔍 CF-Visitor: %s", c.GetHeader("CF-Visitor"))
	log.Printf("🔍 X-Forwarded-Ssl: %s", c.GetHeader("X-Forwarded-Ssl"))

	// Check 1: Direct TLS connection
	if c.Request.TLS != nil {
		scheme = "https"
	}

	// Check 2: Proxy headers (Cloudflare, nginx, etc.)
	forwardedProto := c.GetHeader("X-Forwarded-Proto")
	if forwardedProto == "https" {
		scheme = "https"
	}

	// Check 3: Cloudflare specific header
	cfVisitor := c.GetHeader("CF-Visitor")
	if len(cfVisitor) > 0 && (cfVisitor == `{"scheme":"https"}` || strings.Contains(cfVisitor, `"scheme":"https"`)) {
		scheme = "https"
	}

	// Check 4: Standard forwarded header
	if c.GetHeader("X-Forwarded-Ssl") == "on" {
		scheme = "https"
	}

	// Check 5: If host doesn't have port and not localhost, assume HTTPS (production CDN)
	host := c.Request.Host
	if !strings.Contains(host, ":") && !strings.Contains(host, "localhost") && !strings.Contains(host, "127.0.0.1") {
		scheme = "https"
	}

	log.Printf("✅ Final URL scheme: %s://%s", scheme, host)

	// Return the full image URL using /uploads/ path
	imageURL := fmt.Sprintf("%s://%s/uploads/%s", scheme, host, filename)
	log.Printf("📸 Generated image URL: %s", imageURL)

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"image_url": imageURL,
		"filename":  filename,
	})
}

// DeleteImageHandler deletes an uploaded image file
func DeleteImageHandler(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filename required"})
		return
	}

	// Construct file path
	filepath := filepath.Join("uploads", filename)

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	// Delete the file
	if err := os.Remove(filepath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Image deleted"})
}

// ManageThreeDPageHandler renders the 3D results management page
func ManageThreeDPageHandler(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, date, result, created_at, updated_at 
		FROM threed 
		ORDER BY date DESC
	`)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "manage_threed.html", gin.H{
			"Error": "Failed to fetch 3D results",
		})
		return
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int
		var date, result string
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&id, &date, &result, &createdAt, &updatedAt); err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"ID":        id,
			"Date":      date,
			"Result":    result,
			"CreatedAt": createdAt,
			"UpdatedAt": updatedAt,
		})
	}

	c.HTML(http.StatusOK, "manage_threed.html", gin.H{
		"title":   "Manage 3D Results - Admin",
		"Results": results,
	})
}

// ManagePaperPageHandler renders the paper management page
func ManagePaperPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "manage_paper.html", gin.H{
		"title": "Manage Paper - Admin",
	})
}

// CreateThreeDPageHandler renders the create 3D result form
func CreateThreeDPageHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "create_threed.html", gin.H{
		"title": "Create 3D Result - Admin",
		"Today": time.Now().Format("2006-01-02"),
	})
}

// CreateThreeDHandler handles creating a new 3D result
func CreateThreeDHandler(c *gin.Context) {
	date := c.PostForm("date")
	result := c.PostForm("result")

	// Validate inputs
	if date == "" || result == "" {
		c.HTML(http.StatusBadRequest, "create_threed.html", gin.H{
			"Error": "All fields are required",
			"Today": time.Now().Format("2006-01-02"),
		})
		return
	}

	if len(result) != 3 {
		c.HTML(http.StatusBadRequest, "create_threed.html", gin.H{
			"Error": "Result must be exactly 3 digits",
			"Today": time.Now().Format("2006-01-02"),
		})
		return
	}

	// Insert into database
	query := `
		INSERT INTO threed (date, result, created_at, updated_at) 
		VALUES ($1, $2, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`
	_, err := db.Exec(query, date, result)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "create_threed.html", gin.H{
			"Error": "Failed to create result. Date might already exist.",
			"Today": time.Now().Format("2006-01-02"),
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/threed?message=Result created successfully")
}

// EditThreeDPageHandler renders the edit 3D result form
func EditThreeDPageHandler(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/threed")
		return
	}

	var result struct {
		ID        int
		Date      string
		Result    string
		CreatedAt time.Time
		UpdatedAt time.Time
	}
	var date time.Time

	err = db.QueryRow("SELECT id, date, result, created_at, updated_at FROM threed WHERE id = $1", id).
		Scan(&result.ID, &date, &result.Result, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/threed")
		return
	}

	result.Date = date.Format("2006-01-02")

	c.HTML(http.StatusOK, "edit_threed.html", gin.H{
		"title":  "Edit 3D Result - Admin",
		"Result": result,
	})
}

// EditThreeDHandler handles updating a 3D result
func EditThreeDHandler(c *gin.Context) {
	idStr := c.PostForm("id")
	result := c.PostForm("result")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/threed")
		return
	}

	if len(result) != 3 {
		c.HTML(http.StatusBadRequest, "edit_threed.html", gin.H{
			"Error": "Result must be exactly 3 digits",
		})
		return
	}

	query := `UPDATE threed SET result = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
	_, err = db.Exec(query, result, id)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "edit_threed.html", gin.H{
			"Error": "Failed to update result",
		})
		return
	}

	c.Redirect(http.StatusFound, "/admin/threed?message=Result updated successfully")
}

// DeleteThreeDHandler handles deleting a 3D result
func DeleteThreeDHandler(c *gin.Context) {
	idStr := c.PostForm("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/threed")
		return
	}

	_, err = db.Exec("DELETE FROM threed WHERE id = $1", id)
	if err != nil {
		c.Redirect(http.StatusFound, "/admin/threed?message=Failed to delete result")
		return
	}

	c.Redirect(http.StatusFound, "/admin/threed?message=Result deleted successfully")
}

// ServeImageHandler serves images from the uploads directory via API endpoint
// ServeImageHandler serves images via API endpoint to bypass static file restrictions
func ServeImageHandler(c *gin.Context) {
	filename := c.Param("filename")
	if filename == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filename required"})
		return
	}

	// Get uploads directory - check env variable first, then use relative path
	uploadsDir := os.Getenv("UPLOADS_PATH")
	if uploadsDir == "" {
		uploadsDir = "./uploads"
	}

	// Construct file path
	imagePath := filepath.Join(uploadsDir, filename)

	log.Printf("📸 Serving image: %s (uploads dir: %s, full path: %s)", filename, uploadsDir, imagePath)

	// Check if file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		log.Printf("❌ Image not found: %s", imagePath)
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// NO CACHE - Always serve fresh
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	// Serve the file with appropriate content type (Gin handles this automatically)
	log.Printf("✅ Serving image successfully: %s", filename)
	c.File(imagePath)
}
