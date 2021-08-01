package server

import (
	bytes2 "bytes"
	"context"
	"crypto"
	"errors"
	"fmt"
	"io"
	"strings"

	generatedResponses "github.com/freetocompute/kebe/generated/responses"

	store "github.com/freetocompute/kebe/pkg/store/responses"

	"github.com/freetocompute/kebe/pkg/models"
	"github.com/freetocompute/kebe/pkg/objectstore"
	"github.com/freetocompute/kebe/pkg/sha"
	"github.com/minio/minio-go/v7"

	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"
	"github.com/freetocompute/kebe/pkg/auth"
	"github.com/spf13/viper"

	"gopkg.in/macaroon.v2"
	macaroonv2 "gopkg.in/macaroon.v2"

	"github.com/freetocompute/kebe/pkg/repositories"

	"github.com/sirupsen/logrus"

	"github.com/freetocompute/kebe/pkg/dashboard/requests"
	"github.com/freetocompute/kebe/pkg/dashboard/responses"
	"github.com/freetocompute/kebe/pkg/middleware"
)

type IDashboardHandler interface {
	VerifyACL(verify *requests.Verify) (*responses.Verify, error)
	GetAccount(accountEmail string) (*responses.AccountInfo, error)
	RegisterSnapName(accountEmail string, dryRun bool, snapName string) (*responses.RegisterSnap, error)
	AddAccountKey(accountEmail string, keyName string, publicKeyId string, pubKeyEncoded string) (*models.Key, error)
	GetACLMacaroon(acl string) (*macaroonv2.Macaroon, error)
	GetUploadStatus(upDownId string) (*responses.Status, error)
	PushSnap(snapName string, upDownId string, fileSize uint, channels []string) (*store.Upload, error)
	ReleaseSnap(name string, revision uint, channels []string) (bool, error)
	GetSnapChannelMap(snapName string) (*generatedResponses.Root, error)
}

type DashboardHandler struct {
	accounts repositories.IAccountRepository
	snaps    repositories.ISnapsRepository
}

func NewDashboardHandler(accts repositories.IAccountRepository, snaps repositories.ISnapsRepository) *DashboardHandler {
	return &DashboardHandler{accounts: accts, snaps: snaps}
}

func (d *DashboardHandler) GetSnapChannelMap(snapName string) (*generatedResponses.Root, error) {
	snap, err := d.snaps.GetSnap(snapName, false)
	if err == nil && snap != nil {
		var root generatedResponses.Root
		var channelMapItems []*generatedResponses.ChannelMapItems
		var revisions []*generatedResponses.RevisionsItems
		var channelItems []*generatedResponses.ChannelsItems
		var snapTracks []*generatedResponses.TracksItems

		logrus.Tracef("Getting tracks for: %s", snap.Name)

		tracks, err2 := d.snaps.GetTracks(snap.ID)
		if err2 == nil && tracks != nil {
			for _, track := range *tracks {
				snapTracks = append(snapTracks, &generatedResponses.TracksItems{
					Name: track.Name,
				})

				logrus.Tracef("Getting risks for track: %s", track.Name)

				risks, err3 := d.snaps.GetRisks(track.ID)
				if err3 == nil && risks != nil {
					for _, risk := range *risks {
						logrus.Tracef("Getting revision for risk: %s", risk.Name)
						revision, err4 := d.snaps.GetRevision(risk.RevisionID)
						if err4 == nil && revision != nil {
							logrus.Tracef("Got revision %d", revision.ID)
							channelMapItems = append(channelMapItems, &generatedResponses.ChannelMapItems{
								Architecture: "amd64",
								Channel:      track.Name + "/" + risk.Name,
								Revision:     int(revision.ID),
								Progressive:  &generatedResponses.Progressive{},
							})
							//
							revisions = append(revisions, &generatedResponses.RevisionsItems{
								Architectures: []string{"amd64"},
								Revision:      int(revision.ID),
								Version:       "1",
								Attributes:    &generatedResponses.Attributes{},
								Confinement:   "strict",
								Epoch:         &generatedResponses.Epoch{},
								Grade:         "stable",
								Sha3384:       revision.SHA3_384,
								Size:          int(revision.Size),
							})
							//
							channelItems = append(channelItems, &generatedResponses.ChannelsItems{
								Name:  track.Name + "/" + risk.Name,
								Risk:  risk.Name,
								Track: track.Name,
							})
						}
					}
				}
			}

			root.ChannelMap = channelMapItems
			root.Revisions = revisions

			root.Snap = &generatedResponses.Snap{
				Channels: channelItems,
				Name:     snap.Name,
				Tracks:   snapTracks,
			}
			return &root, nil
		}
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	unknownErr := errors.New("unknown error encountered")
	logrus.Error(unknownErr)

	panic("unknown error encountered")
}

func (d *DashboardHandler) ReleaseSnap(name string, revision uint, channels []string) (bool, error) {
	if name != "" && revision != 0 && len(channels) > 0 {
		snapEntry, err := d.snaps.GetSnap(name, false)
		if err == nil && snapEntry != nil {
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
					logrus.Error(errors.New("branches not supported yet"))
					return false, errors.New("branches not supported yet")
				}

				track, err2 := d.snaps.SetChannelRevision(trackForRelease, riskForRelease, revision, snapEntry.ID)
				if err2 != nil {
					logrus.Error(err2)
					return false, err2
				}

				if track == nil {
					logrus.Errorf("could not set revision for track/risk: %s/%s", trackForRelease, riskForRelease)
					return false, errors.New("could not set revision for track")
				}
			}

			return true, nil

		} else if err != nil {
			logrus.Error(err)
			return false, err
		}
	} else {
		emptyErr := errors.New("all fields must be non-empty")
		logrus.Error(emptyErr)
		return false, emptyErr
	}

	unknownError := errors.New("unknown error encountered")
	logrus.Error(unknownError)
	return false, unknownError
}

func (d *DashboardHandler) PushSnap(snapName string, upDownId string, fileSize uint, channels []string) (*store.Upload, error) {
	snapUpload, err := d.snaps.AddUpload(snapName, upDownId, fileSize, channels)
	if err == nil && snapUpload != nil {
		//// File saved successfully. Return proper result
		//// TODO: this URL needs to be serviced by a worker thread
		snapUploadResp := store.Upload{
			Success: true,
			// TODO: check this at start-up, verify Must then
			StatusDetailsURL: config.MustGetString(configkey.DashboardURL) + "/dev/api/snap-status/" + upDownId,
		}

		return &snapUploadResp, nil
	}

	return nil, errors.New("unknown error encountered")
}

func (d *DashboardHandler) GetUploadStatus(upDownId string) (*responses.Status, error) {
	// We need to move the snap from the unscanned bucket to the snaps bucket
	snapUpload, err := d.snaps.GetUpload(upDownId)
	if err == nil && snapUpload != nil {
		snapFileName := upDownId + ".snap"

		// get the sha3_384 of the file so we can figure out if it already exists as a revision
		obj, err2 := objectstore.GetMinioClient().GetObject(context.Background(), "unscanned", snapFileName, minio.GetObjectOptions{})
		if err2 != nil {
			panic(err2)
		}

		bytes, err3 := io.ReadAll(obj)
		h := crypto.SHA3_384.New()
		if err3 != nil {
			panic(err3)
		}
		h.Write(bytes)
		actualSha3 := fmt.Sprintf("%x", h.Sum(nil))

		rev, err4 := d.snaps.GetRevisionBySHA(actualSha3, false)
		var revision models.SnapRevision
		if err4 == nil && rev != nil {
			revision = *rev
			logrus.Infof("Revision %s found to exist for snap %s, updating channels with existing revision", actualSha3, snapUpload.Name)
			// This revision already exists on some channel, we just
			// need to update the requested channels to have this revision

			// need to discard upload. remove record
			// TODO: add to auditing later?
			logrus.Infof("Removing object %s from buckect %s", snapFileName, "unscanned")
			err4 = objectstore.GetMinioClient().RemoveObject(context.Background(), "unscanned", snapFileName, minio.RemoveObjectOptions{})
			if err4 != nil {
				logrus.Error(err4)
			}
		} else if err4 == nil && rev == nil {
			logrus.Infof("Revision %s not found to exist for snap %s, creating revision and updating channels with revision", actualSha3, snapUpload.Name)
			objStore := objectstore.NewObjectStore()
			err4 = objStore.Move("unscanned", "snaps", snapFileName)
			if err4 != nil {
				logrus.Error(err4)
				return nil, err4
			}

			digest, _, err5 := sha.SnapFileSHA3_384FromReader(bytes2.NewReader(bytes))
			if err5 != nil {
				panic(err5)
			}

			revision = models.SnapRevision{
				SnapFilename:   snapFileName,
				SnapEntryID:    snapUpload.SnapEntryID,
				SHA3_384:       actualSha3,
				SHA3384Encoded: digest,
				Size:           int64(snapUpload.Filesize),
			}

			_, err2 = d.snaps.UpdateRevision(&revision, &bytes)
			if err2 != nil {
				logrus.Error(err2)
				return nil, err2
			}
		} else {
			logrus.Error(err4)
			return nil, err4
		}

		// TODO: fix lazy
		channels := strings.Split(snapUpload.Channels, ",")
		err2 = d.snaps.ReleaseSnap(channels, snapUpload.SnapEntryID, revision.ID)
		if err2 == nil {
			resp := &responses.Status{
				Processed: true,
				Code:      "ready_to_release",
				Revision:  int(revision.ID),
			}

			return resp, nil
		}

		logrus.Error(err2)
		return nil, err2
	}

	logrus.Error(err)
	return nil, err
}

func (d *DashboardHandler) GetACLMacaroon(acl string) (*macaroonv2.Macaroon, error) {
	// TODO: check these sooner, cache the values and ensure they exist on start-up
	rootKeyString := config.MustGetString(configkey.MacaroonRootKey)
	rootMacaroonId := config.MustGetString(configkey.MacaroonRootId)
	rootMacaroonLocation := config.MustGetString(configkey.MacaroonRootLocation)
	m := auth.MustNewMacaroon([]byte(rootKeyString), []byte(rootMacaroonId), rootMacaroonLocation, macaroon.V1)

	dischargeKeyString := viper.GetString(configkey.MacaroonDischargeKey)
	if len(dischargeKeyString) == 0 {
		// this is panic worthy
		// TODO: check these sooner
		panic(errors.New("discharge key must be set"))
	}
	thirdPartyCaveatId := config.MustGetString(configkey.MacaroonThirdPartyCaveatId)
	thirdPartLocation := config.MustGetString(configkey.MacaroonThirdPartyLocation)
	err := m.AddThirdPartyCaveat([]byte(dischargeKeyString), []byte(thirdPartyCaveatId), thirdPartLocation)
	if err != nil {
		panic(err)
	}

	_ = m.AddFirstPartyCaveat([]byte(acl))

	return m, nil
}

func (d *DashboardHandler) AddAccountKey(accountEmail string, keyName string, publicKeyId string, pubKeyEncoded string) (*models.Key, error) {
	if accountEmail != "" && keyName != "" && publicKeyId != "" && pubKeyEncoded != "" {
		acct, err2 := d.accounts.AddKey(keyName, publicKeyId, pubKeyEncoded, accountEmail)
		if err2 == nil && acct != nil {
			return acct, nil
		} else if err2 != nil {
			logrus.Error(err2)
			return nil, err2
		}
	}

	logrus.Errorf("accountEmail=%s, keyName=%s, publicKeyId=%s and pubKeyEncoded=%s must all be non-empty", accountEmail, keyName, publicKeyId, pubKeyEncoded)
	return nil, errors.New("email, key name, public key id and public key encoded must all be non-empty")
}

func (d *DashboardHandler) RegisterSnapName(accountEmail string, isDryRun bool, snapName string) (*responses.RegisterSnap, error) {
	if accountEmail != "" {
		account, err2 := d.accounts.GetAccountByEmail(accountEmail, false)
		if err2 == nil && account != nil {
			if !isDryRun {
				snap, err3 := d.snaps.AddSnap(snapName, account.ID)
				if err3 == nil && snap != nil {
					resp := responses.RegisterSnap{
						Id:   snap.SnapStoreID,
						Name: snap.Name,
					}

					return &resp, nil
				}
			} else {
				snap, err3 := d.snaps.GetSnap(snapName, false)
				if err3 == nil && snap != nil {
					resp := responses.RegisterSnap{
						Name: snapName,
					}
					return &resp, nil
				}
			}
		}
	}

	return nil, errors.New("account not found")
}

func (d *DashboardHandler) GetAccount(accountEmail string) (*responses.AccountInfo, error) {
	if accountEmail != "" {
		account, err := d.accounts.GetAccountByEmail(accountEmail, true)

		if err == nil {
			accountInfoResponse := responses.AccountInfo{
				AccountId:   account.AccountId,
				Snaps:       map[string]map[string]responses.Snap{},
				AccountKeys: []responses.Key{},
			}

			for _, k := range account.Keys {
				accountInfoResponse.AccountKeys = append(accountInfoResponse.AccountKeys, responses.Key{
					PublicKeySHA384: k.SHA3384,
					Name:            k.Name,
				})
			}

			snaps := map[string]responses.Snap{}
			for _, s := range account.SnapEntries {
				// TODO: replace with real data
				snaps[s.Name] = responses.Snap{
					Status:  "Approved",
					SnapId:  s.SnapStoreID,
					Store:   "Global",
					Since:   "2016-07-04T23:37:52Z",
					Private: false,
				}
			}

			accountInfoResponse.Snaps["16"] = snaps

			// TODO: this would actually need to be filled in
			return &accountInfoResponse, nil
		}

		logrus.Error(err)
	}

	return nil, errors.New("not found")
}

func (d *DashboardHandler) VerifyACL(verify *requests.Verify) (*responses.Verify, error) {
	userEmail, err := middleware.VerifyAndGetEmail(verify.AuthData.Authorization)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	user, err := d.accounts.GetAccountByEmail(*userEmail, false)
	if err != nil {
		logrus.Error(err)
		return nil, err
	}

	if user != nil {
		v := responses.Verify{
			Allowed:               true,
			DeviceRefreshRequired: false,
			RefreshRequired:       false,
			Account: &responses.VerifyAccount{
				Email:       user.Email,
				DisplayName: user.DisplayName,
				OpenId:      "oid1234",
				Verified:    true,
			},
			Device:      nil,
			LastAuth:    "2016-05-26T12:53:23Z",
			Permissions: &[]string{"package_access", "package_manage", "package_push", "package_register", "package_release", "package_update"},
			SnapIds:     nil,
			Channels:    nil,
		}

		return &v, nil
	}

	return nil, errors.New("user not found")
}
