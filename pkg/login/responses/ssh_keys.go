package responses

type SSHKeys struct {
	Username         string   `json:"username"`
	SSHKeys          []string `json:"ssh_keys"`
	OpenIdIdentifier string   `json:"openid_identifier"`
}
