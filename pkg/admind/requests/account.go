package requests

type AddAccount struct {
	Username    string
	AcccountId  string
	Email       string
	DisplayName string
}

type AddTrack struct {
	SnapName  string
	TrackName string
}
