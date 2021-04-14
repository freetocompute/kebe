package responses

import "github.com/snapcore/snapd/snap"

// storeSnap holds the information sent as JSON by the store for a snap.
type StoreSnap struct {
	Architectures []string          `json:"architectures"`
	Base          *string           `json:"base"`
	Confinement   string            `json:"confinement"`
	Contact       string            `json:"contact"`
	CreatedAt     string            `json:"created-at"` // revision timestamp
	Description   string            `json:"description"`
	Download      StoreSnapDownload `json:"download"`
	Epoch         snap.Epoch        `json:"epoch"`
	License       string            `json:"license"`
	Name          string            `json:"name"`
	Prices        map[string]string `json:"prices"` // currency->price,  free: {"USD": "0"}
	Private       bool              `json:"private"`
	Publisher     snap.StoreAccount `json:"publisher"`
	Revision      int               `json:"revision"` // store revisions are ints starting at 1
	SnapID        string            `json:"snap-id"`
	SnapYAML      string            `json:"snap-yaml"` // optional
	Summary       string            `json:"summary"`
	Title         string            `json:"title"`
	Type          snap.Type         `json:"type"`
	Version       string            `json:"version"`
	Website       string            `json:"website"`
	StoreURL      string            `json:"store-url"`

	// TODO: not yet defined: channel map

	// media
	Media []StoreSnapMedia `json:"media"`

	CommonIDs []string `json:"common-ids"`
}
