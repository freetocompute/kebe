package repositories

import (
	"errors"
	"strings"

	"gorm.io/gorm/clause"

	"github.com/freetocompute/kebe/pkg/snap"
	"github.com/sirupsen/logrus"

	"github.com/google/uuid"

	"github.com/freetocompute/kebe/pkg/database"
	"github.com/freetocompute/kebe/pkg/models"
	"gorm.io/gorm"
)

type ISnapsRepository interface {
	GetSnap(name string, preloadAssociations bool) (*models.SnapEntry, error)
	AddSnap(name string, accountId uint) (*models.SnapEntry, error)

	GetRevisionBySHA(SHA3_384 string) (*models.SnapRevision, error)
	GetUpload(upDownId string) (*models.SnapUpload, error)
	UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, error)

	ReleaseSnap(channels []string, snapEntryId uint, revisionId uint) error
	AddUpload(snapName string, upDownId string, size uint, channels []string) (*models.SnapUpload, error)

	SetChannelRevision(trackName string, riskName string, revisionId uint, snapId uint) (*models.SnapTrack, error)

	GetTracks(snapId uint) (*[]models.SnapTrack, error)
	GetRisks(trackId uint) (*[]models.SnapRisk, error)
	GetRevision(id uint) (*models.SnapRevision, error)

	GetSections() (*[]string, error)

	GetSnaps() (*[]models.SnapEntry, error)
}

type SnapsRepository struct {
	db *gorm.DB
}

func NewSnapsRepository(db *gorm.DB) *SnapsRepository {
	return &SnapsRepository{db: db}
}

func (sp *SnapsRepository) GetSections() (*[]string, error) {
	// TODO: add these to the database for real
	sections := []string{
		"general",
	}

	return &sections, nil
}

func (sp *SnapsRepository) GetTracks(snapId uint) (*[]models.SnapTrack, error) {
	var tracks []models.SnapTrack
	db := sp.db.Where(&models.SnapTrack{SnapEntryID: snapId}).Find(&tracks)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &tracks, nil
	}

	if db.Error != nil {
		return nil, db.Error
	}

	logrus.Errorf("Could not find tracks for snapId: %d", snapId)
	return nil, errors.New("unknown error encountered")
}

func (sp *SnapsRepository) GetRisks(trackId uint) (*[]models.SnapRisk, error) {
	var risks []models.SnapRisk
	db := sp.db.Where(&models.SnapRisk{SnapTrackID: trackId}).Find(&risks)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &risks, nil
	}

	if db.Error != nil {
		return nil, db.Error
	}

	logrus.Errorf("Could not find risks for track id: %d", trackId)
	return nil, errors.New("unknown error encountered")
}

func (sp *SnapsRepository) GetRevision(id uint) (*models.SnapRevision, error) {
	var revision models.SnapRevision
	db := sp.db.Where(&models.SnapRevision{Model: gorm.Model{ID: id}}).Find(&revision)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &revision, nil
	}

	if db.Error != nil {
		return nil, db.Error
	}

	return nil, errors.New("unknown error encountered")
}

func (sp *SnapsRepository) SetChannelRevision(trackName string, riskName string, revisionId uint, snapId uint) (*models.SnapTrack, error) {
	// get all the tracks
	var track models.SnapTrack
	db := sp.db.Where(&models.SnapTrack{SnapEntryID: snapId, Name: trackName}).Find(&track)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		// get all the risks
		var risk models.SnapRisk
		db = sp.db.Where(&models.SnapRisk{SnapEntryID: snapId, Name: riskName, SnapTrackID: track.ID}).Find(&risk)
		if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
			var revision models.SnapRevision
			db = sp.db.Where("id", revisionId).Find(&revision)
			if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
				risk.RevisionID = revision.ID
				sp.db.Save(&risk)
				return &track, nil
			}
		} else {
			return nil, errors.New("risk does not exist for track")
		}
	} else {
		return nil, errors.New("track does not exist for snap")
	}

	return nil, errors.New("unknown error encountered")
}

func (sp *SnapsRepository) AddUpload(snapName string, upDownId string, fileSize uint, channels []string) (*models.SnapUpload, error) {
	var snap models.SnapEntry
	db := sp.db.Where(&models.SnapEntry{Name: snapName}).Find(&snap)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		snapUpload := models.SnapUpload{
			Name:        snapName,
			UpDownID:    upDownId,
			Filesize:    fileSize,
			SnapEntryID: snap.ID,
		}

		logrus.Infof("Uploading: %+v", snapUpload)

		// TODO: fix lazy; this should be converted to a table so that the channels can be stored separately or maybe redis
		if len(channels) > 0 {
			channelsString := ""
			for _, chn := range channels {
				if channelsString == "" {
					channelsString = chn
				} else {
					channelsString = channelsString + "," + chn
				}
			}

			snapUpload.Channels = channelsString
		}

		db2 := sp.db.Save(&snapUpload)
		if _, ok := database.CheckDBForErrorOrNoRows(db2); ok {
			return &snapUpload, nil
		}
	}

	if db.Error != nil {
		logrus.Error(db.Error)
		return nil, db.Error
	}

	return nil, errors.New("unknown error encountered")
}

func (sp *SnapsRepository) GetRevisionBySHA(SHA3_384 string) (*models.SnapRevision, error) {
	var revision models.SnapRevision
	db := sp.db.Where(models.SnapRevision{SHA3_384: SHA3_384}).Find(&revision)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &revision, nil
	} else if db.Error == nil {
		return nil, nil
	}

	return nil, db.Error
}

func (sp *SnapsRepository) GetUpload(upDownId string) (*models.SnapUpload, error) {
	var snapUpload models.SnapUpload
	db := sp.db.Where(&models.SnapUpload{UpDownID: upDownId}).Find(&snapUpload)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &snapUpload, nil
	}

	return nil, errors.New("not found")
}

func (sp *SnapsRepository) UpdateRevision(revision *models.SnapRevision, revisionBytes *[]byte) (*models.SnapRevision, error) {
	db := sp.db.Save(revision)
	if db.Error == nil {
		err := sp.updateMeta(revisionBytes)
		if err == nil {
			return revision, nil
		}
	}
	return nil, db.Error
}

func (sp *SnapsRepository) GetSnaps() (*[]models.SnapEntry, error) {
	var snaps []models.SnapEntry

	// TODO: would need to implement private and filter here
	db := sp.db.Find(&snaps)
	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &snaps, nil
	}

	// TODO: evaluate this, we shouldn't ever really have _NO_ snaps
	// It's not an error to find no snaps... or is it?
	if db.Error == nil {
		return &snaps, nil
	}

	return nil, db.Error
}

func (sp *SnapsRepository) GetSnap(name string, preloadAssociations bool) (*models.SnapEntry, error) {
	var existingSnap models.SnapEntry
	var db *gorm.DB
	if preloadAssociations {
		db = sp.db.Preload(clause.Associations).Where(&models.SnapEntry{Name: name}).Find(&existingSnap)
	} else {
		db = sp.db.Where(&models.SnapEntry{Name: name}).Find(&existingSnap)
	}

	if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
		return &existingSnap, nil
	}

	if db.Error != nil {
		return nil, db.Error
	}

	logrus.Errorf("Could not find snap %s", name)

	return nil, db.Error
}

func (sp *SnapsRepository) AddSnap(name string, accountId uint) (*models.SnapEntry, error) {
	existingSnap, err := sp.GetSnap(name, false)
	if err == nil && existingSnap != nil {
		// when adding a snap, not finding one _is_ (!ok) what you want
		var newSnapEntry models.SnapEntry
		snapId := uuid.New()
		newSnapEntry.SnapStoreID = snapId.String()
		newSnapEntry.Name = name
		newSnapEntry.AccountID = accountId
		newSnapEntry.Type = "app"

		sp.db.Save(&newSnapEntry)

		// For now when we register a snap we are going to create the default tracks/risks
		track := models.SnapTrack{
			Name:        "latest",
			SnapEntryID: newSnapEntry.ID,
		}

		sp.db.Save(&track)

		sp.addRisks(newSnapEntry.ID, track.ID)

		return &newSnapEntry, nil
	}

	return nil, errors.New("there was an error")
}

func (sp *SnapsRepository) AddDefaultRisks(snapEntryId uint, trackId uint) {
	sp.addRisks(snapEntryId, trackId)
}

func (sp *SnapsRepository) ReleaseSnap(channels []string, snapEntryId uint, revisionId uint) error {
	var trackForRelease string
	var riskForRelease string
	for _, cn := range channels {
		// It's possible this comes in the form:
		//   - single string values "edge" where the track is assumed to be "latest" there is no branch
		//   - two values "latest/edge" where the risk is proceeded by the track
		//   - three values "latest/edge/some_branch"
		parts := strings.Split(cn, "/")
		if len(parts) == 1 {
			riskForRelease = parts[0]
			trackForRelease = "latest"
		} else if len(parts) == 2 {
			trackForRelease = parts[0]
			riskForRelease = parts[1]
		} else if len(parts) == 3 {
			return errors.New("branches not supported yet")
		}

		// get all the tracks
		var track models.SnapTrack
		db := sp.db.Where(&models.SnapTrack{SnapEntryID: snapEntryId, Name: trackForRelease}).Find(&track)
		if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
			// get all the risks
			var risk models.SnapRisk
			db = sp.db.Where(&models.SnapRisk{SnapEntryID: snapEntryId, Name: riskForRelease, SnapTrackID: track.ID}).Find(&risk)
			if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
				var revision models.SnapRevision
				db = sp.db.Where("id", revisionId).Find(&revision)
				if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
					risk.RevisionID = revision.ID
					sp.db.Save(&risk)
				}
			}
		}
	}

	return nil
}

func (sp *SnapsRepository) addRisks(snapEntryId uint, trackId uint) {
	// TODO: fix me
	risks := []string{"stable", "candidate", "beta", "edge"}

	// TODO: fix the need for an empty revision
	snapRevision := models.SnapRevision{
		SnapFilename: "",
		SnapEntryID:  snapEntryId,
		SHA3_384:     "",
		Size:         0,
	}

	sp.db.Save(&snapRevision)

	for _, risk := range risks {
		var snapRisk models.SnapRisk
		snapRisk.SnapEntryID = snapEntryId
		snapRisk.SnapTrackID = trackId
		snapRisk.Name = risk

		snapRisk.RevisionID = snapRevision.ID

		sp.db.Save(&snapRisk)
	}
}

func (sp *SnapsRepository) updateMeta(metaBytes *[]byte) error {
	snapMeta, err2 := snap.GetSnapMetaFromBytes(*metaBytes, "/tmp")
	if err2 == nil {
		logrus.Tracef("snapMeta: %+v", snapMeta)
		var snapEntry models.SnapEntry
		db := sp.db.Where(&models.SnapEntry{Name: snapMeta.Name}).Find(&snapEntry)
		if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
			snapEntry.Type = "app"
			if snapMeta.Type != "" {
				snapEntry.Type = snapMeta.Type
			} else {
				logrus.Warnf("Snap %s had an emtpy type from its metadata, using default '%s'", snapEntry.Name, snapEntry.Type)
			}

			snapEntry.Confinement = snapMeta.Confinement
			snapEntry.Base = snapMeta.Base

			sp.db.Save(&snapEntry)
		} else {
			logrus.Errorf("No rows found for: %s", snapMeta.Name)
		}
	}

	return err2
}
