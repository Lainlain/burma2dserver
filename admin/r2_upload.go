package admin

import (
	"context"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// R2Client holds the Cloudflare R2 client configuration
type R2Client struct {
	client     *s3.Client
	bucketName string
	publicURL  string
	enabled    bool
}

var r2Client *R2Client

// InitR2 initializes the Cloudflare R2 client with credentials from environment
func InitR2() error {
	// Check if R2 is enabled
	useR2 := os.Getenv("USE_R2")
	if useR2 != "true" {
		log.Println("📦 R2 upload disabled (USE_R2 != true), using local storage")
		r2Client = &R2Client{enabled: false}
		return nil
	}

	accountID := os.Getenv("R2_ACCOUNT_ID")
	accessKeyID := os.Getenv("R2_ACCESS_KEY_ID")
	secretAccessKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucketName := os.Getenv("R2_BUCKET_NAME")
	publicURL := os.Getenv("R2_PUBLIC_URL")

	// Validate required credentials
	if accountID == "" || accessKeyID == "" || secretAccessKey == "" || bucketName == "" {
		return fmt.Errorf("missing R2 credentials: R2_ACCOUNT_ID, R2_ACCESS_KEY_ID, R2_SECRET_ACCESS_KEY, R2_BUCKET_NAME required")
	}

	// Build R2 endpoint (S3-compatible)
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	// If no public URL provided, use R2 endpoint (will need custom domain in production)
	if publicURL == "" {
		publicURL = endpoint
		log.Printf("⚠️  R2_PUBLIC_URL not set, using R2 endpoint (not publicly accessible)")
	}

	// Create AWS config with R2 credentials
	r2Config := aws.Config{
		Region: "auto", // R2 uses "auto" region
		Credentials: credentials.NewStaticCredentialsProvider(
			accessKeyID,
			secretAccessKey,
			"", // Session token not needed
		),
	}

	// Create S3 client pointing to R2 endpoint
	s3Client := s3.NewFromConfig(r2Config, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
		o.UsePathStyle = true // Required for R2
	})

	r2Client = &R2Client{
		client:     s3Client,
		bucketName: bucketName,
		publicURL:  publicURL,
		enabled:    true,
	}

	log.Printf("✅ Cloudflare R2 initialized: bucket=%s, endpoint=%s", bucketName, endpoint)
	return nil
}

// IsR2Enabled returns whether R2 upload is enabled
func IsR2Enabled() bool {
	return r2Client != nil && r2Client.enabled
}

// UploadToR2 uploads a file to Cloudflare R2 and returns the public URL
func UploadToR2(file *multipart.FileHeader) (string, error) {
	if !IsR2Enabled() {
		return "", fmt.Errorf("R2 client not initialized or disabled")
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Generate unique filename with timestamp
	ext := filepath.Ext(file.Filename)
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("gifts/%d_%s%s", timestamp, filepath.Base(file.Filename[:len(file.Filename)-len(ext)]), ext)

	// Detect content type
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = detectContentType(ext)
	}

	log.Printf("📤 Uploading to R2: bucket=%s, key=%s, size=%d bytes", r2Client.bucketName, filename, file.Size)

	// Upload to R2
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	_, err = r2Client.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(r2Client.bucketName),
		Key:           aws.String(filename),
		Body:          src,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(file.Size),
		// Make object publicly readable (if bucket has public access)
		// ACL: types.ObjectCannedACLPublicRead, // R2 doesn't support ACLs, use bucket settings
	})

	if err != nil {
		return "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	// Build public URL
	publicURL := fmt.Sprintf("%s/%s", r2Client.publicURL, filename)

	log.Printf("✅ R2 upload successful: %s", publicURL)
	return publicURL, nil
}

// detectContentType returns MIME type based on file extension
func detectContentType(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// DeleteFromR2 deletes a file from R2 (optional cleanup function)
func DeleteFromR2(fileURL string) error {
	if !IsR2Enabled() {
		return fmt.Errorf("R2 client not initialized or disabled")
	}

	// Extract key from URL (assumes format: https://domain/key)
	// Example: https://pub-xxx.r2.dev/gifts/1234567890_gift.jpg -> gifts/1234567890_gift.jpg
	key := filepath.Base(fileURL)
	if filepath.Dir(fileURL) != "." {
		key = filepath.Join(filepath.Base(filepath.Dir(fileURL)), key)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := r2Client.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r2Client.bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete from R2: %w", err)
	}

	log.Printf("🗑️  Deleted from R2: %s", key)
	return nil
}
