package requests

type RegisterSnapName struct {
	Name    string `json:"snap_name"`
	Private bool   `json:"is_private"`
	Store   string `json:"store"`
}
