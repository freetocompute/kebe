package responses

import (
	store2 "github.com/snapcore/snapd/store"
)

type SearchV2Results struct {
	Results   []StoreSearchResult `json:"results"`
	ErrorList []struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error-list"`
}

// storeSearchChannelSnap is the snap revision plus a channel name
type StoreSearchChannelSnap struct {
	StoreSnap
	Channel string `json:"channel"`
}

// storeSearchResult is the result of v2/find calls
type StoreSearchResult struct {
	Revision StoreSearchChannelSnap `json:"revision"`
	Snap     StoreSnap              `json:"snap"`
	Name     string                 `json:"name"`
	SnapID   string                 `json:"snap-id"`
}

type StoreSnapDownload struct {
	Sha3_384 string           `json:"sha3-384"`
	Size     int64            `json:"size"`
	URL      string           `json:"url"`
	Deltas   []StoreSnapDelta `json:"deltas"`
}

type StoreSnapDelta struct {
	Format   string `json:"format"`
	Sha3_384 string `json:"sha3-384"`
	Size     int64  `json:"size"`
	Source   int    `json:"source"`
	Target   int    `json:"target"`
	URL      string `json:"url"`
}

type StoreSnapMedia struct {
	Type   string `json:"type"` // icon/screenshot
	URL    string `json:"url"`
	Width  int64  `json:"width"`
	Height int64  `json:"height"`
}

type Payload struct {
	Sections []Section `json:"clickindex:sections"`
}

type Section struct {
	Name string
}

type SectionResults struct {
	Payload Payload `json:"_embedded"`
}

type Alias struct {
	Name string `json:"name"`
}

type CatalogPayload struct {
	Items []CatalogItem `json:"clickindex:package"`
}

type CatalogResults struct {
	Payload CatalogPayload `json:"_embedded"`
}

type CatalogItem struct {
	Name    string   `json:"package_name"`
	Version string   `json:"version"`
	Summary string   `json:"summary"`
	Aliases []Alias  `json:"aliases"`
	Apps    []string `json:"apps"`
	Title   string   `json:"title"`
}

type SnapRelease struct {
	Architecture string `json:"architecture"`
	Channel      string `json:"channel"`
}

type ErrorListEntry struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	// for assertions
	Type       string   `json:"type"`
	PrimaryKey []string `json:"primary-key"`
}

type SnapActionResultList struct {
	Results   []*SnapActionResult `json:"results"`
	ErrorList []ErrorListEntry    `json:"error-list"`
}

type SnapActionResultListRedux struct {
	Results   []*store2.SnapActionResult `json:"results"`
	ErrorList []ErrorListEntry           `json:"error-list"`
}

type SnapActionExtra struct {
	Releases []SnapRelease `json:"releases"`
}

type SnapActionResultError struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Extra   SnapActionExtra `json:"extra"`
}

type SnapActionResult struct {
	Result string `json:"result"`
	// For snap
	InstanceKey      string                `json:"instance-key"`
	SnapID           string                `json:"snap-id,omitempty"`
	Name             string                `json:"name,omitempty"`
	Snap             *StoreSnap            `json:"snap"`
	EffectiveChannel string                `json:"effective-channel,omitempty"`
	RedirectChannel  string                `json:"redirect-channel,omitempty"`
	Error            SnapActionResultError `json:"error"`
	// For assertions
	Key                 string           `json:"key"`
	AssertionStreamURLs []string         `json:"assertion-stream-urls"`
	ErrorList           []ErrorListEntry `json:"error-list"`
}

type Unscanned struct {
	UploadId string `json:"upload_id"`
}

type Upload struct {
	Success          bool
	StatusDetailsURL string `json:"status_details_url"`
}

type Nonce struct {
	Nonce string `json:"nonce"`
}

type Session struct {
	Macaroon string `json:"macaroon"`
}

type AuthRequestIDResp struct {
	RequestID string `json:"request-id"`
}
