package responses

type Macaroon struct {
	Macaroon string `json:"macaroon"`
}

type DischargeMacaroon struct {
	DischargeMacaroon string `json:"discharge_macaroon"`
}

type VerifyAccount struct {
	Email string `json:"email"`
	DisplayName string `json:"displayname"`
	OpenId string `json:"openid"`
	Verified bool `json:"verified"`
}

type Verify struct {
	Allowed bool `json:"allowed"`
	DeviceRefreshRequired bool `json:"device_refresh_required"`
	RefreshRequired bool `json:"refresh_required"`
	Account *VerifyAccount `json:"account,omitempty"`
	Device *string `json:"device"`
	LastAuth string `json:"last_auth"`
	Permissions *[]string `json:"permissions"`
	SnapIds *string `json:"snap_ids"`
	Channels *string `json:"channels"`
}