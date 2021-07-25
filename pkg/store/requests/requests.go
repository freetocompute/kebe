package requests

import (
	"github.com/snapcore/snapd/snap"
	"time"
)

type RequestIDResp struct {
	RequestID string `json:"request-id"`
}

type SnapActionRequest struct {
	Context             []*CurrentSnapV2JSON `json:"context"`
	Actions             []*SnapActionJSON    `json:"actions"`
	Fields              []string             `json:"fields"`
	AssertionMaxFormats map[string]int       `json:"assertion-max-formats,omitempty"`
}

type CurrentSnapV2JSON struct {
	SnapID           string     `json:"snap-id"`
	InstanceKey      string     `json:"instance-key"`
	Revision         int        `json:"revision"`
	TrackingChannel  string     `json:"tracking-channel"`
	Epoch            snap.Epoch `json:"epoch"`
	RefreshedDate    *time.Time `json:"refreshed-date,omitempty"`
	IgnoreValidation bool       `json:"ignore-validation,omitempty"`
	CohortKey        string     `json:"cohort-key,omitempty"`
}

type SnapActionJSON struct {
	Action string `json:"action"`
	// For snap
	InstanceKey      string `json:"instance-key,omitempty"`
	Name             string `json:"name,omitempty"`
	SnapID           string `json:"snap-id,omitempty"`
	Channel          string `json:"channel,omitempty"`
	Revision         int    `json:"revision,omitempty"`
	CohortKey        string `json:"cohort-key,omitempty"`
	IgnoreValidation *bool  `json:"ignore-validation,omitempty"`

	// NOTE the store needs an epoch (even if null) for the "install" and "download"
	// actions, to know the client handles epochs at all.  "refresh" actions should
	// send nothing, not even null -- the snap in the context should have the epoch
	// already.  We achieve this by making Epoch be an `interface{}` with omitempty,
	// and then setting it to a (possibly nil) epoch for install and download. As a
	// nil epoch is not an empty interface{}, you'll get the null in the json.
	Epoch interface{} `json:"epoch,omitempty"`
	// For assertions
	Key        string         `json:"key,omitempty"`
	Assertions []AssertAtJSON `json:"assertions,omitempty"`
}

type AssertAtJSON struct {
	Type        string   `json:"type"`
	PrimaryKey  []string `json:"primary-key"`
	IfNewerThan *int     `json:"if-newer-than,omitempty"`
}

type SnapPush struct {
	Name           string
	DryRun         bool   `json:"dry_run"`
	UpDownId       string `json:"updown_id"`
	Series         string
	BinaryFileSize int64    `json:"binary_filesize"`
	SourceUploaded bool     `json:"source_uploaded"`
	DeltaFormat    string   `json:"delta_format"`
	DeltaHash      string   `json:"delta_hash"`
	SourceHash     string   `json:"source_hash"`
	TargetHash     string   `json:"target_hash"`
	Channels       []string `json:"channels"`
}

type SnapRelease struct {
	Name     string
	Revision string
	Channels []string
}

type Session struct {
	// asserts.DeviceSessionRequest
	DeviceSessionRequest string `json:"device-session-request"`
	ModelAssertion       string `json:"model-assertion"`
	SerialAssertion      string `json:"serial-assertion"`
}
