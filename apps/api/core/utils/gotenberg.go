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

// PDFOptions controls the page format Gotenberg renders the HTML into.
type PDFOptions struct {
	Landscape   bool
	PaperWidth  string // inches
	PaperHeight string // inches
}

// A5Landscape is the paper format used by the prescription PDF (kept as the
// default for GeneratePDFFromHTML for backwards compatibility).
var A5Landscape = PDFOptions{Landscape: true, PaperWidth: "8.3", PaperHeight: "5.8"}

// A4Portrait is the standard paper format for a RIDE (comprobante SRI) PDF.
var A4Portrait = PDFOptions{Landscape: false, PaperWidth: "8.27", PaperHeight: "11.7"}

// GeneratePDFFromHTML sends an HTML string to Gotenberg and returns the generated PDF
// bytes, using the A5-landscape format (prescription PDF default).
func (g *GotenbergClient) GeneratePDFFromHTML(htmlContent string) ([]byte, error) {
	return g.GeneratePDFFromHTMLWithOptions(htmlContent, A5Landscape)
}

// GeneratePDFFromHTMLWithOptions sends an HTML string to Gotenberg and returns the
// generated PDF bytes, using the given page format.
func (g *GotenbergClient) GeneratePDFFromHTMLWithOptions(htmlContent string, opts PDFOptions) ([]byte, error) {
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

	landscape := "false"
	if opts.Landscape {
		landscape = "true"
	}
	if err := writer.WriteField("landscape", landscape); err != nil {
		return nil, fmt.Errorf("failed to write landscape field: %w", err)
	}
	if err := writer.WriteField("paperWidth", opts.PaperWidth); err != nil {
		return nil, fmt.Errorf("failed to write paperWidth: %w", err)
	}
	if err := writer.WriteField("paperHeight", opts.PaperHeight); err != nil {
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
