package requests

type Discharge struct {
	Email string
	Password string
	CaveatId string `json:"caveat_id"`
}