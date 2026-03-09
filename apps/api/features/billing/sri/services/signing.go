package services

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
)

// getP12FromLocalFile reads a P12 file from the local filesystem
func GetP12FromLocalFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// getP12FromUrl fetches a P12 file from a URL
func GetP12FromUrl(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// getXMLFromLocalFile reads an XML file from the local filesystem
func GetXMLFromLocalFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// getXMLFromLocalUrl fetches an XML file from a URL
func GetXMLFromLocalUrl(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// sha1Base64 computes the SHA1 hash of a string and returns it as a base64-encoded string
func Sha1Base64(text string) string {
	hasher := sha1.New()
	hasher.Write([]byte(text))
	hash := hasher.Sum(nil)
	return base64.StdEncoding.EncodeToString(hash)
}

// hexToBase64 converts a hex string to a base64-encoded string
func HexToBase64(hexStr string) string {
	bytes, _ := hex.DecodeString(hexStr)
	return base64.StdEncoding.EncodeToString(bytes)
}

// bigIntToBase64 converts a big integer to a base64-encoded string
func BigIntToBase64(bigInt *big.Int) string {
	bytes := bigInt.Bytes()
	return base64.StdEncoding.EncodeToString(bytes)
}

// getRandomNumber generates a random number between min and max
func GetRandomNumber(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return int(n.Int64()) + min
}

func SignXml(p12Data []byte, p12Password string, xmlData string) (string, error) {
	payload := map[string]string{
		"xml_data_base64": base64.StdEncoding.EncodeToString([]byte(xmlData)),
		"p12_data_base64": base64.StdEncoding.EncodeToString(p12Data),
		"password_base64": base64.StdEncoding.EncodeToString([]byte(p12Password)),
	}

	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post("http://signer:8001/firmar_base64", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error POST to signer service: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error from signer service: %s", string(body))
	}

	var result struct {
		SignedXmlBase64 string `json:"signed_xml_base64"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(result.SignedXmlBase64)
	if err != nil {
		return "", fmt.Errorf("error decoding signed xml: %w", err)
	}

	return string(decoded), nil
}
