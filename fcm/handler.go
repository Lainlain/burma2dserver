package fcm

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// NotificationRequest represents the request body for sending notifications
type NotificationRequest struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body" binding:"required"`
}

// SendNotificationHandler handles sending custom notifications from admin
func SendNotificationHandler(c *gin.Context) {
	var req NotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Send notification to gifts topic
	if err := SendCustomNotification(req.Title, req.Body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send notification",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Notification sent to gifts topic",
		"title":   req.Title,
		"body":    req.Body,
	})
}
