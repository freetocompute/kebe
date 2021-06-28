package responses

type Macaroon struct {
	Macaroon string `json:"macaroon"`
}

type DischargeMacaroon struct {
	DischargeMacaroon string `json:"discharge_macaroon"`
}