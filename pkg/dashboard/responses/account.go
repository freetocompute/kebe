package responses

type AccountInfo struct {
	Snaps map[string]map[string]map[string]string `json:"snaps"`
	AccountKeys []Key `json:"account_keys"`
	AccountId string `json:"account_id"`
}

type Key struct {
	PublicKeySHA384 string `json:"public-key-sha3-384"`
	Name string `json:"name"`
}
