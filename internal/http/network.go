package http

import (
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// fetchImageAsBase64 fetches an image from a URL and returns it as a base64-encoded data URI
// Returns an empty string if any errors occur
func FetchImageAsBase64(imageURL string) string {
	// Set a timeout for the HTTP client
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Make the HTTP request
	resp, err := client.Get(imageURL)
	if err != nil {
		log.Printf("Error fetching image %s: %v\n", imageURL, err)
		return ""
	}
	defer resp.Body.Close()

	// Check if response status is OK
	if resp.StatusCode != http.StatusOK {
		log.Printf("Error fetching image %s: status code %d\n", imageURL, resp.StatusCode)
		return ""
	}

	// Read the image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading image data %s: %v\n", imageURL, err)
		return ""
	}

	// Determine content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		// Try to guess from URL extension
		if strings.HasSuffix(strings.ToLower(imageURL), ".jpg") || strings.HasSuffix(strings.ToLower(imageURL), ".jpeg") {
			contentType = "image/jpeg"
		} else if strings.HasSuffix(strings.ToLower(imageURL), ".png") {
			contentType = "image/png"
		} else if strings.HasSuffix(strings.ToLower(imageURL), ".gif") {
			contentType = "image/gif"
		} else if strings.HasSuffix(strings.ToLower(imageURL), ".webp") {
			contentType = "image/webp"
		} else {
			contentType = "image/jpeg" // Default assumption
		}
	}

	// Encode the image data as base64
	base64Encoded := base64.StdEncoding.EncodeToString(imageData)

	// Create the data URI
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64Encoded)

	return dataURI
}
