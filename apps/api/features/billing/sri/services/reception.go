package services

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"

	"io"
	"net/http"
)

type Envelope struct {
	XMLName xml.Name `xml:"Envelope"`
	Body    Body     `xml:"Body"`
}

type EnvelopeAutorization struct {
	XMLName xml.Name         `xml:"Envelope"`
	Body    BodyAutorization `xml:"Body"`
}

type Body struct {
	Response ValidarComprobanteResponse `xml:"validarComprobanteResponse"`
}

type BodyAutorization struct {
	Response ValidarComprobanteResponseAutorization `xml:"autorizacionComprobanteResponse"`
}

type ValidarComprobanteResponse struct {
	Respuesta RespuestaRecepcionComprobante `xml:"RespuestaRecepcionComprobante"`
}

type ValidarComprobanteResponseAutorization struct {
	Respuesta RespuestaAutorizacionComprobante `xml:"RespuestaAutorizacionComprobante"`
}

type RespuestaRecepcionComprobante struct {
	Estado       string        `xml:"estado"`
	Comprobantes []Comprobante `xml:"comprobantes>comprobante"`
}

type RespuestaAutorizacionComprobante struct {
	Authorizations []Autorizacion `xml:"autorizaciones>autorizacion"`
}

type Autorizacion struct {
	Estado             string      `xml:"estado"`
	NumeroAutorizacion string      `xml:"numeroAutorizacion"`
	FechaAutorizacion  string      `xml:"fechaAutorizacion"`
	Ambiente           string      `xml:"ambiente"`
	Comprobante        Comprobante `xml:"comprobante"`
	Mensajes           []Mensaje   `xml:"mensajes>mensaje"`
}

type Comprobante struct {
	ClaveAcceso string    `xml:"claveAcceso"`
	Mensajes    []Mensaje `xml:"mensajes>mensaje"`
}

type Mensaje struct {
	Identificador        string `xml:"identificador"`
	Mensaje              string `xml:"mensaje"`
	InformacionAdicional string `xml:"informacionAdicional"`
	Tipo                 string `xml:"tipo"`
}

func DocumentReception(xml []byte, receptionURL string) (*RespuestaRecepcionComprobante, error) {
	base64XML := base64.StdEncoding.EncodeToString(xml)

	soapEnvelope := fmt.Sprintf(`
		<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/"
						  xmlns:ec="http://ec.gob.sri.ws.recepcion">
			<soapenv:Header/>
			<soapenv:Body>
				<ec:validarComprobante>
					<xml>%s</xml>
				</ec:validarComprobante>
			</soapenv:Body>
		</soapenv:Envelope>`, base64XML)

	req, err := http.NewRequest("POST", receptionURL, bytes.NewBufferString(soapEnvelope))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "text/xml;charset=UTF-8")
	req.Header.Add("SOAPAction", "") // No se necesita para SRI

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return ParseSOAPResponse(data)
}

func ParseSOAPResponse(response []byte) (*RespuestaRecepcionComprobante, error) {
	var envelope Envelope
	if err := xml.Unmarshal(response, &envelope); err != nil {
		return nil, fmt.Errorf("error unmarshalling XML: %w", err)
	}
	return &envelope.Body.Response.Respuesta, nil
}
