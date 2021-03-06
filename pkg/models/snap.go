package models

import (
	"context"
	"crypto"
	"fmt"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/objectstore"
	"github.com/freetocompute/kebe/pkg/store/responses"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"github.com/snapcore/snapd/snap"
	"github.com/spf13/viper"
	"gorm.io/gorm"
	"io"
)

type SnapTrack struct {
	gorm.Model
	Name string

	SnapEntryID uint
	SnapEntry   SnapEntry

	Risks []SnapRisk
}

type SnapRisk struct {
	gorm.Model
	Name        string
	SnapTrackID uint
	SnapEntryID uint
	SnapEntry   SnapEntry

	// TODO: fix this -- currently this is monotonically incrementing across ALL revisions, it should just be a given snap
	RevisionID uint
	Revision   SnapRevision

	Branches []SnapBranch
}

type SnapBranch struct {
	gorm.Model
	Name        string
	SnapRiskID  uint
	SnapEntryID uint
	SnapEntry   SnapEntry

	RevisionID uint
	Revision   SnapRevision
}

type SnapEntry struct {
	gorm.Model
	Name        string `json:"name"`
	SnapStoreID string `json:"snap-id"`
	Revisions   []SnapRevision
	Type        string
	Confinement string
	Base        string
	Uploads     []SnapUpload

	AccountID uint
	Account   Account
}

type SnapRevision struct {
	gorm.Model
	SnapFilename   string
	SnapEntryID    uint
	SHA3_384       string
	SHA3384Encoded string `gorm:"column:sha3_384_encoded"`
	Size           int64
}

type SnapUpload struct {
	gorm.Model
	Name     string
	UpDownID string
	Filesize uint
	// Channels is a comma-separated string of channels
	Channels    string
	SnapEntryID uint
	SnapEntry   SnapEntry
}

func (se *SnapEntry) ToStoreSnap(snapRevision *SnapRevision) (*responses.StoreSnap, error) {
	downloadURL := fmt.Sprintf(viper.GetString(configkey.StoreAPIURL)+"/download/snaps/%s", snapRevision.SnapFilename)
	base := snapRevision.SnapFilename
	obs := objectstore.NewObjectStore()
	h := crypto.SHA3_384.New()
	objectPtr, err := obs.MinioClient.GetObject(context.Background(), "snaps", base, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	bytes, _ := io.ReadAll(objectPtr)
	h.Write(bytes)
	actualSha3 := fmt.Sprintf("%x", h.Sum(nil))

	logrus.Infof("Snap: %s, Revision: %d, URL: %s, SHA3: %s", se.Name, snapRevision.ID, downloadURL, actualSha3)

	storeSnap := &responses.StoreSnap{
		Name:     se.Name,
		Type:     snap.Type(se.Type),
		SnapID:   se.SnapStoreID,
		Revision: int(snapRevision.ID),
		Download: responses.StoreSnapDownload{
			Sha3_384: actualSha3,
			Size:     snapRevision.Size,
			URL:      downloadURL,
		},
		Confinement: se.Confinement,
		Base:        &se.Base,
	}

	return storeSnap, nil
}
