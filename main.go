package main

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"burma2d/admin"
	"burma2d/fcm"
	"burma2d/gift"
	"burma2d/live"
	"burma2d/paper"
	"burma2d/slider"
	"burma2d/threed"
	"burma2d/twodhistory"

	"github.com/gin-gonic/gin"
)

func main() {
	// Set umask to 0022 so files are created with correct permissions
	// This means new files will be 644 and directories 755
	syscall.Umask(0022)

	// Create Gin router
	r := gin.Default()

	// Enable CORS for all origins
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Initialize database
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		// Default SQLite database file
		dbPath = "./burma2d.db"
	}

	log.Printf("🔌 Attempting database connection...")
	log.Printf("� Database file: %s", dbPath)

	dbEnabled := false
	if err := twodhistory.InitDB(dbPath); err != nil {
		log.Printf("❌ Database initialization failed: %v", err)
		log.Println("⚠️  Continuing without database features...")
		log.Println("⚠️  Admin routes and data APIs will not be available!")
	} else {
		defer twodhistory.CloseDB()
		dbEnabled = true
		log.Println("✅ Database connected successfully!")

		// Initialize gift and slider packages
		db := twodhistory.GetDB()
		gift.InitDB(db)
		slider.InitDB(db)
		admin.InitDB(db)
		threed.InitDB(db)
		paper.InitDB(db)
		log.Println("✅ All database modules initialized!")
	}

	// Initialize live package
	live.Init()

	// Initialize Firebase Cloud Messaging
	firebasePath := "./dexpect-2be84-firebase-adminsdk-fbsvc-520abe0b4f.json"
	if err := fcm.InitFCM(firebasePath); err != nil {
		log.Printf("⚠️ Warning: Firebase FCM initialization failed: %v", err)
		log.Println("⚠️ Gift notifications will not be sent")
	}

	// Register history inserter callback if database is enabled
	if dbEnabled {
		live.SetHistoryInserter(func(data *live.LotteryData) error {
			// Convert live.LotteryData to twodhistory.LotteryData
			histData := &twodhistory.LotteryData{
				Date:        data.Date,
				Live:        data.Live,
				Status:      data.Status,
				Set1200:     data.Set1200,
				Value1200:   data.Value1200,
				Result1200:  data.Result1200,
				Set430:      data.Set430,
				Value430:    data.Value430,
				Result430:   data.Result430,
				Modern930:   data.Modern930,
				Internet930: data.Internet930,
				Modern200:   data.Modern200,
				Internet200: data.Internet200,
				UpdateTime:  data.UpdateTime,
			}
			return twodhistory.InsertFromLotteryData(histData)
		})
		log.Println("✅ History auto-insert enabled (16:30-16:35 GMT+6:30)")
	}

	// Routes - Game API (rebranded from lottery)
	r.POST("/api/game/update", live.UpdateLotteryData)
	r.GET("/api/game/stream", live.StreamLotteryData)
	r.GET("/api/game/current", live.GetCurrentData)

	// History routes (rebranded)
	r.GET("/api/game/history", twodhistory.GetHistoryHandler)
	r.POST("/api/game/history/check", twodhistory.CheckAndInsertHandler)

	// Rewards routes (rebranded from gifts)
	r.GET("/api/game/rewards", gift.GetGiftsHandler)

	// Banners routes (rebranded from sliders)
	r.GET("/api/game/banners", slider.GetSlidersHandler)

	// 3D routes
	r.GET("/api/game/3d", threed.GetAllResults)
	r.POST("/api/game/3d", threed.CreateResult)
	r.PUT("/api/game/3d", threed.UpdateResult)
	r.DELETE("/api/game/3d", threed.DeleteResult)

	// Guides routes (rebranded from paper)
	r.GET("/api/game/guides/types", paper.GetAllTypes)
	r.GET("/api/game/guides/types/:type_id/images", paper.GetImagesByType)

	// Image serving route - static files from uploads directory
	r.Static("/uploads", "./uploads")

	// Admin routes
	if dbEnabled {
		// Load HTML templates
		r.LoadHTMLGlob("admin/templates/*.html")

		// Admin dashboard pages
		r.GET("/admin", admin.AdminDashboardHandler)
		r.GET("/admin/gifts", admin.ManageGiftsPageHandler)
		r.GET("/admin/sliders", admin.ManageSlidersPageHandler)
		r.GET("/admin/threed", admin.ManageThreeDPageHandler)
		r.GET("/admin/paper", admin.ManagePaperPageHandler)
		r.GET("/admin/gifts/create", admin.CreateGiftPageHandler)
		r.GET("/admin/sliders/create", admin.CreateSliderPageHandler)
		r.GET("/admin/threed/create", admin.CreateThreeDPageHandler)
		r.POST("/admin/threed/create", admin.CreateThreeDHandler)
		r.GET("/admin/gifts/edit/:id", admin.EditGiftPageHandler)
		r.GET("/admin/sliders/edit/:id", admin.EditSliderPageHandler)
		r.GET("/admin/threed/edit", admin.EditThreeDPageHandler)
		r.POST("/admin/threed/edit", admin.EditThreeDHandler)
		r.POST("/admin/threed/delete", admin.DeleteThreeDHandler)

		// Image upload routes
		r.POST("/api/admin/upload-image", admin.UploadImageHandler)
		r.DELETE("/api/admin/delete-image/:filename", admin.DeleteImageHandler)

		// Version/Health check endpoint
		r.GET("/api/version", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"version": "1.0.0",
				"service": "Burma 2D 2025 API",
			})
		})

		// Admin API routes for gifts
		r.GET("/api/admin/gifts", func(c *gin.Context) {
			gifts, err := gift.GetAllGiftsForAdmin()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gifts)
		})
		r.GET("/api/admin/gifts/:id", admin.GetGiftByIDHandler)
		r.POST("/api/admin/gifts", func(c *gin.Context) {
			var newGift gift.Gift
			if err := c.BindJSON(&newGift); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			if err := gift.InsertGift(newGift); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Gift created"})
		})
		r.PUT("/api/admin/gifts/:id", func(c *gin.Context) {
			var updatedGift gift.Gift
			if err := c.BindJSON(&updatedGift); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			if err := gift.UpdateGift(updatedGift); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Gift updated"})
		})
		r.DELETE("/api/admin/gifts/:id", func(c *gin.Context) {
			var id int
			if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
				c.JSON(400, gin.H{"error": "Invalid ID"})
				return
			}
			if err := gift.DeleteGift(id); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Gift deleted"})
		})

		// Admin API routes for sliders
		r.GET("/api/admin/sliders", func(c *gin.Context) {
			sliders, err := slider.GetAllSlidersForAdmin()
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, sliders)
		})
		r.GET("/api/admin/sliders/:id", admin.GetSliderByIDHandler)
		r.POST("/api/admin/sliders", func(c *gin.Context) {
			var newSlider slider.Slider
			if err := c.BindJSON(&newSlider); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			if err := slider.InsertSlider(newSlider); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Slider created"})
		})
		r.PUT("/api/admin/sliders/:id", func(c *gin.Context) {
			var updatedSlider slider.Slider
			if err := c.BindJSON(&updatedSlider); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			if err := slider.UpdateSlider(updatedSlider); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Slider updated"})
		})
		r.DELETE("/api/admin/sliders/:id", func(c *gin.Context) {
			var id int
			if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
				c.JSON(400, gin.H{"error": "Invalid ID"})
				return
			}
			if err := slider.DeleteSlider(id); err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"message": "Slider deleted"})
		})

		// Admin API routes for paper
		r.GET("/api/admin/paper/types", paper.GetAllTypesWithImages)
		r.POST("/api/admin/paper/types", paper.CreateType)
		r.PUT("/api/admin/paper/types/:id", paper.UpdateType)
		r.DELETE("/api/admin/paper/types/:id", paper.DeleteType)
		r.POST("/api/admin/paper/images", paper.CreateImage)
		r.POST("/api/admin/paper/images/batch", paper.BatchCreateImages)
		r.PUT("/api/admin/paper/images/:id", paper.UpdateImage)
		r.DELETE("/api/admin/paper/images/:id", paper.DeleteImage)
	}

	// Privacy Policy route (public)
	r.GET("/privacy-policy", func(c *gin.Context) {
		c.HTML(200, "privacy-policy.html", gin.H{})
	})

	// Landing page
	r.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", nil)
	})

	// Start server
	log.Println("🚀 Server starting on :4545")
	log.Println("📡 SSE Stream available at: http://localhost:4545/api/game/stream")
	log.Println("📮 POST game data to: http://localhost:4545/api/game/update")
	log.Println("📜 History data at: http://localhost:4545/api/game/history")
	if err := r.Run(":4545"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
