package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

type APIResponse[T any] struct {
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type SriSignerClient struct {
	httpClient *http.Client
	baseURL    string
}

type SriSignRequest struct {
	P12Buffer []byte `json:"p12Buffer"`
	Password  string `json:"password"`
	XMLBuffer []byte `json:"xmlBuffer"`
}

type SriSignResponse struct {
	XML string `json:"xml"`
}

func NewSriSignerClient() *SriSignerClient {
	url := os.Getenv("SRI_SIGNER_SERVICE_URL")
	if url == "" {
		url = "http://sri-xml-signer:9000"
	}
	return &SriSignerClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    url,
	}
}

func (ssc *SriSignerClient) Post(endpoint string, payload any) (*http.Response, error) {
	url := fmt.Sprintf("%s/%s", ssc.baseURL, endpoint)

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return ssc.httpClient.Do(req)
}

func (ssc *SriSignerClient) SignXML(p12Buffer []byte, password string, xmlBuffer []byte) (*SriSignResponse, error) {
	payload := SriSignRequest{
		P12Buffer: p12Buffer,
		Password:  password,
		XMLBuffer: xmlBuffer,
	}
	var apiResp APIResponse[SriSignResponse]
	resp, err := ssc.Post("sign", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&apiResp)
		return nil, fmt.Errorf("%s", apiResp.Message)
	}
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		return nil, err
	}

	return &apiResp.Data, nil
}

func (ssc *SriSignerClient) ValidateXMLWithSRI(xmlBuffer []byte, env string) (map[string]any, error) {
	payload := map[string]any{
		"xml": xmlBuffer,
	}
	var apiResp APIResponse[map[string]any]
	resp, err := ssc.Post(fmt.Sprintf("validate/%s", env), payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&apiResp)
		return nil, fmt.Errorf("%s", apiResp.Message)
	}

	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (ssc *SriSignerClient) AuthorizeXMLWithSRI(accessCode string, env string) (map[string]any, error) {
	payload := map[string]any{
		"accessKey": accessCode,
	}
	var apiResp APIResponse[map[string]any]
	resp, err := ssc.Post(fmt.Sprintf("authorization/%s", env), payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		json.NewDecoder(resp.Body).Decode(&apiResp)
		return nil, fmt.Errorf("%s", apiResp.Message)
	}

	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (ssc *SriSignerClient) SignXMLMultipart(p12 []byte, password string, xml []byte) (*SriSignResponse, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("p12", "cert.p12")
	if err != nil {
		return nil, err
	}
	if _, err := fw.Write(p12); err != nil {
		return nil, err
	}

	fw2, err := w.CreateFormFile("xml", "invoice.xml")
	if err != nil {
		return nil, err
	}
	if _, err := fw2.Write(xml); err != nil {
		return nil, err
	}

	if err := w.WriteField("password", password); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	url := ssc.baseURL + "/sign"
	req, err := http.NewRequest(http.MethodPost, url, &b)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := ssc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp APIResponse[SriSignResponse]
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return &apiResp.Data, nil
}
