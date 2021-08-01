package admind

import "golang.org/x/oauth2"

type Token struct {
	AccessToken      string   `json:"access_token"`
	ExpiresIn        int32    `json:"expires_in"`
	RefreshExpiresIn int32    `json:"refresh_expires_in"`
	RefreshToken     string   `json:"refresh_token"`
	TokenType        string   `json:"token_type"`
	IdToken          string   `json:"id_token"`
	Scope            []string `json:"scope"`
}

type UserInfo struct {
	Sub               string   `json:"sub"`
	EmailVerified     string   `json:"email_verified"`
	Name              string   `json:"name"`
	Groups            []string `json:"groups"`
	PreferredUsername string   `json:"preferred_username"`
	GivenName         string   `json:"given_name"`
	FamilyName        string   `json:"family_name"`
	Email             string   `json:"email"`
}

type LoginInfo struct {
	UserInfo UserInfo     `json:"user_info"`
	Token    oauth2.Token `json:"token"`
}
