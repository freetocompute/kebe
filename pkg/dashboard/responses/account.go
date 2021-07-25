package responses

type Snap struct {
	Status  string `json:"status"`
	SnapId  string `json:"snap-id"`
	Store   string `json:"store"`
	Since   string `json:"since"`
	Private bool   `json:"private"`
	Price   string `json:"price"`
}

type AccountInfo struct {
	Snaps       map[string]map[string]Snap `json:"snaps"`
	AccountKeys []Key                      `json:"account_keys"`
	AccountId   string                     `json:"account_id"`
}

type Key struct {
	PublicKeySHA384 string `json:"public-key-sha3-384"`
	Name            string `json:"name"`
}
