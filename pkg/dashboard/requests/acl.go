package requests

type ACLRequest struct {
	Permissions []string
	Channels []string
	Packages []string
	Expires string
}

