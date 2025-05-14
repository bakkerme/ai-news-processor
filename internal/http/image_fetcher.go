package http

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ImageFetcher interface {
	FetchAsBase64(imageURL string) (string, error)
}

// DefaultImageFetcher is the default implementation of imagefetcher.ImageFetcher
type DefaultImageFetcher struct{}

// FetchAsBase64 fetches an image from a URL and returns it as a base64-encoded data URI.
// It implements the imagefetcher.ImageFetcher interface.
// The original logic from FetchImageAsBase64 is moved here.
// It now returns an error instead of an empty string on failure.
func (dif *DefaultImageFetcher) FetchAsBase64(imageURL string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("error fetching image %s: %w", imageURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error fetching image %s: status code %d", imageURL, resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading image data %s: %w", imageURL, err)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
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

	base64Encoded := base64.StdEncoding.EncodeToString(imageData)
	dataURI := fmt.Sprintf("data:%s;base64,%s", contentType, base64Encoded)

	return dataURI, nil
}
