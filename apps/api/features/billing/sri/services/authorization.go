package services

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

// Envelope y Body para estructurar la solicitud SOAP
type soapEnvelope struct {
	XMLName xml.Name     `xml:"soapenv:Envelope"`
	SoapEnv string       `xml:"xmlns:soapenv,attr"`
	Ws      string       `xml:"xmlns:ws,attr"`
	Body    soapBodyAuth `xml:"soapenv:Body"`
}

type soapBodyAuth struct {
	Request AuthRequest `xml:"ws:autorizacionComprobante"`
}

type AuthRequest struct {
	ClaveAcceso string `xml:"claveAccesoComprobante"`
}

// Aquí puedes definir la estructura del XML de respuesta
// Para simplificar, se deja como interface{} pero puedes definirlo correctamente.
func DocumentAuthorization(accessKey, authorizationURL string) (interface{}, error) {
	// Crear la estructura SOAP XML de la petición
	envelope := soapEnvelope{
		SoapEnv: "http://schemas.xmlsoap.org/soap/envelope/",
		Ws:      "http://ec.gob.sri.ws.autorizacion",
		Body: soapBodyAuth{
			Request: AuthRequest{
				ClaveAcceso: accessKey,
			},
		},
	}

	// Serializar la estructura a XML
	payload, err := xml.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling XML: %w", err)
	}

	// Agregar cabeceras SOAP necesarias
	req, err := http.NewRequest("POST", authorizationURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Content-Type", "text/xml; charset=utf-8")
	req.Header.Add("SOAPAction", "") // Ecuador SRI no requiere un valor aquí

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SOAP request failed: %s", string(bodyBytes))
	}

	// Leer la respuesta completa (puedes usar estructuras para parsear si deseas)
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	response, _ := parseSOAPResponse(bodyBytes)

	fmt.Println(string(bodyBytes))

	for _, autorizacion := range response.Authorizations {
		for _, mensaje := range autorizacion.Mensajes {
			if mensaje.Tipo == "ERROR" || mensaje.Tipo == "ADVERTENCIA" {
				return nil, fmt.Errorf("%s", mensaje.Mensaje)
			}

		}
	}

	// Aquí puedes usar una estructura definida si quieres parsear el XML
	return string(bodyBytes), nil
}

func parseSOAPResponse(response []byte) (*RespuestaAutorizacionComprobante, error) {
	var envelope EnvelopeAutorization
	if err := xml.Unmarshal(response, &envelope); err != nil {
		return nil, fmt.Errorf("error unmarshalling XML: %w", err)
	}
	return &envelope.Body.Response.Respuesta, nil
}
