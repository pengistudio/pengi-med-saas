package utils

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

type GotenbergClient struct {
	URL string
}

func NewGotenbergClient(url string) *GotenbergClient {
	return &GotenbergClient{URL: url}
}

// GeneratePDFFromHTML sends an HTML string to Gotenberg and returns the generated PDF bytes.
func (g *GotenbergClient) GeneratePDFFromHTML(htmlContent string) ([]byte, error) {
	// Gotenberg endpoint for HTML to PDF
	endpoint := fmt.Sprintf("%s/forms/chromium/convert/html", g.URL)

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)

	// Add the main HTML file
	part, err := writer.CreateFormFile("files", "index.html")
	if err != nil {
		return nil, fmt.Errorf("failed to create form file for index.html: %w", err)
	}

	_, err = io.WriteString(part, htmlContent)
	if err != nil {
		return nil, fmt.Errorf("failed to write html content to form: %w", err)
	}

	// Add landscape setting if needed (Gotenberg uses 'landscape' form field)
	err = writer.WriteField("landscape", "true")
	if err != nil {
		return nil, fmt.Errorf("failed to write landscape field: %w", err)
	}

	// Set paper size to A5 (148 x 210 mm = 5.8 x 8.3 inches)
	err = writer.WriteField("paperWidth", "8.3")
	if err != nil {
		return nil, fmt.Errorf("failed to write paperWidth: %w", err)
	}
	err = writer.WriteField("paperHeight", "5.8")
	if err != nil {
		return nil, fmt.Errorf("failed to write paperHeight: %w", err)
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", endpoint, &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request to Gotenberg at %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gotenberg responded with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Read PDF bytes
	pdfBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read pdf response body: %w", err)
	}

	return pdfBytes, nil
}
