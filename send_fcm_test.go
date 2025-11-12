package main

import (
	"log"

	"burma2d/fcm"
)

func main() {
	serviceAccountPath := "../burma2d-67734-firebase-adminsdk-fbsvc-f40c69cacd.json" // Update path if needed
	if err := fcm.InitFCM(serviceAccountPath); err != nil {
		log.Fatalf("Failed to initialize FCM: %v", err)
	}

	giftName := "Test Gift"
	if err := fcm.SendGiftAvailableNotification(giftName); err != nil {
		log.Fatalf("Failed to send FCM notification: %v", err)
	}

	log.Println("✅ Test FCM notification sent to 'gifts' topic!")
}
