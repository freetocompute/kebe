package models

import (
	"fmt"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/store/responses"
	"github.com/snapcore/snapd/snap"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type SnapEntry struct {
	gorm.Model
	Name        string `json:"name"`
	SnapStoreID string `json:"snap-id"`
	//Snap Snap
	LatestRevisionID uint
	Revisions        []SnapRevision
	Type             string
	Confinement      string

	AccountID uint
	Account   Account
}

func (se *SnapEntry) GetLatestRevision() *SnapRevision {
	for _, r := range se.Revisions {
		if r.ID == se.LatestRevisionID {
			return &r
		}
	}

	return nil
}

type SnapRevision struct {
	gorm.Model

	SnapFilename           string
	BuildAssertionFilename string

	SnapEntryID uint
	// SnapEntry SnapEntry
	SHA3_384 string
	Size     int64
}

func (se *SnapEntry) ToStoreSnap(revisionFileName string, size int64, sha3384 string) (*responses.StoreSnap, error) {
	downloadURL := fmt.Sprintf(viper.GetString(configkey.StoreAPIURL)+"/download/snaps/%s", revisionFileName)

	return &responses.StoreSnap{
		Name:     se.Name,
		Type:     snap.Type(se.Type),
		SnapID:   se.SnapStoreID,
		Revision: int(se.LatestRevisionID),
		Download: responses.StoreSnapDownload{
			Sha3_384: sha3384,
			Size:     size,
			URL:      downloadURL,
		},
		Confinement: se.Confinement,
	}, nil
}
