package sri

type Arrival struct {
	TransferReason   string `json:"motivoTraslado" xml:"motivoTraslado"`
	UniqueCustomsDoc string `json:"docAduaneroUnico" xml:"docAduaneroUnico"`
	DestinationCode  string `json:"codEstabDestino" xml:"codEstabDestino"`
	Route            string `json:"ruta" xml:"ruta"`
}

type Arrivals struct {
	Destination []Arrival `json:"destino" xml:"destino"`
}

type RemissionGuideSubstituteInfo struct {
	DepartureAddress              string   `json:"dirPartida" xml:"dirPartida"`
	DestinationAddress            string   `json:"dirDestinatario" xml:"dirDestinatario"`
	TransportStartDate            string   `json:"fechaIniTransporte" xml:"fechaIniTransporte"`
	TransportEndDate              string   `json:"fechaFinTransporte" xml:"fechaFinTransporte"`
	TransporterBusinessName       string   `json:"razonSocialTransportista" xml:"razonSocialTransportista"`
	TransporterIdentificationType string   `json:"tipoIdentificacionTransportista" xml:"tipoIdentificacionTransportista"`
	TransporterRUC                string   `json:"rucTransportista" xml:"rucTransportista"`
	LicensePlate                  string   `json:"placa" xml:"placa"`
	Destinations                  Arrivals `json:"destinos" xml:"destinos"`
}
