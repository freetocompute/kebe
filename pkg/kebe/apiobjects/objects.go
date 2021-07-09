package apiobjects

type Key struct {
	AccountId string `json:"account-id"`
	Name string `json:"name"`
	SHA3384 string `json:"sha3-384"`
	EncodedPublicKey string `json:"encoded-public-key"`
}

type Snap struct {
	Name string `json:"name"`
	SnapStoreID string `json:"snap-id"`
	DeveloperID string `json:"developer-id"`
	Type string `json:"type"`
}

type Account struct {
	DisplayName string `json:"display-name"`
	Username string `json:"username"`
	Keys []Key `json:"keys"`
}
