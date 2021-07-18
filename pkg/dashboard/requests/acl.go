package requests

type ACLRequest struct {
	Permissions []string
	Channels []string
	Packages []string
	Expires string
}

type AuthData struct {
	Authorization string `json:"authorization"`
}

type Verify struct {
	AuthData AuthData `json:"auth_data"`
}