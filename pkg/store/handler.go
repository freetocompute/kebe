package store

import (
	"crypto/rsa"
	"encoding/base64"
	"errors"

	"github.com/snapcore/snapd/asserts/assertstest"

	asserts2 "github.com/freetocompute/kebe/pkg/store/asserts"

	"github.com/freetocompute/kebe/config"
	"github.com/freetocompute/kebe/config/configkey"

	"github.com/snapcore/snapd/asserts"

	"github.com/freetocompute/kebe/pkg/objectstore"

	"github.com/freetocompute/kebe/pkg/store/requests"

	"github.com/snapcore/snapd/snap"

	"github.com/sirupsen/logrus"

	"github.com/freetocompute/kebe/pkg/repositories"
	"github.com/freetocompute/kebe/pkg/store/responses"
)

type IStoreHandler interface {
	GetSections() (*responses.SectionResults, error)
	GetSnapNames() (*responses.CatalogResults, error)
	FindSnap(name string) (*responses.SearchV2Results, error)
	SnapRefresh(actions *[]*requests.SnapActionJSON) (*responses.SnapActionResultList, error)
	SnapDownload(snapFilename string) (*[]byte, error)
	GetSnapRevisionAssertion(SHA3384Encoded string, rootStoreKey *rsa.PrivateKey, assertsDB *asserts.Database) (*asserts.SnapRevision, error)
	GetSnapDeclarationAssertion(snapId string, rootStoreKey *rsa.PrivateKey, assertsDB *asserts.Database) (*asserts.SnapDeclaration, error)
	GetAccountKeyAssertion(keySHA3384 string, rootStoreKey *rsa.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.AccountKey, error)
	GetAccountAssertion(accountId string, rootStoreKey *rsa.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.Account, error)
}

type Handler struct {
	accounts repositories.IAccountRepository
	snaps    repositories.ISnapsRepository
}

func NewHandler(accts repositories.IAccountRepository, snaps repositories.ISnapsRepository) *Handler {
	return &Handler{
		accts,
		snaps,
	}
}

func (h *Handler) GetAccountKeyAssertion(keySHA3384 string, rootStoreKey *rsa.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.AccountKey, error) {
	accountKey, err := h.accounts.GetKeyBySHA3384(keySHA3384)
	if err == nil && accountKey != nil {
		logrus.Tracef("Found account-key: %+v", accountKey)

		bytes, err2 := base64.StdEncoding.DecodeString(accountKey.EncodedPublicKey)
		if err2 != nil {
			panic(err2)
		}

		pbk, err2 := asserts.DecodePublicKey([]byte(bytes))
		if err2 != nil {
			panic(err2)
		}

		trustedAcct := getTrustedAccount(accountKey.Account.AccountId, signingDB, accountKey.Account.DisplayName)

		// TODO: what do do about these dates?
		trustedAcctKeyHeaders := map[string]interface{}{
			"since":               "2015-11-20T15:04:00Z",
			"until":               "2500-11-20T15:04:00Z",
			"public-key-sha3-384": accountKey.SHA3384,
			"name":                accountKey.Name,
		}
		//
		trustedAccKey := assertstest.NewAccountKey(signingDB, trustedAcct, trustedAcctKeyHeaders, pbk, "")
		if trustedAccKey != nil {
			return trustedAccKey, nil
		}
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return nil, errors.New("account key could not be found or there was an error")
}

func (h *Handler) GetAccountAssertion(accountId string, rootStoreKey *rsa.PrivateKey, signingDB *assertstest.SigningDB) (*asserts.Account, error) {
	account, err := h.accounts.GetAccountById(accountId, false)
	if err == nil && account != nil {
		//
		pk := asserts.RSAPrivateKey(rootStoreKey)
		acct := createAccountAssertion(signingDB, pk.PublicKey().ID(), account.AccountId, account.Username)
		return acct, nil
	} else if err != nil {
		return nil, err
	}

	logrus.Errorf("Unknown error, could not find account: %s", accountId)
	return nil, errors.New("account not found")
}

func (h *Handler) GetSnapDeclarationAssertion(snapStoreId string, rootStoreKey *rsa.PrivateKey, assertsDB *asserts.Database) (*asserts.SnapDeclaration, error) {
	logrus.Tracef("Requested snap-declaration: %s", snapStoreId)

	snapEntry, err := h.snaps.GetSnapByStoreId(snapStoreId, true)
	if err == nil && snapEntry != nil {
		// TODO: do this sooner, like during construction to fail then if not MUST
		rootAuthorityId := config.MustGetString(configkey.RootAuthority)

		aaa, err2 := asserts2.MakeSnapDeclarationAssertion(rootAuthorityId, snapEntry.Account.AccountId, snapEntry, asserts.RSAPrivateKey(rootStoreKey), assertsDB)
		if err2 == nil && aaa != nil {
			return aaa, nil
		} else if err2 == nil {
			logrus.Error(err2)
			return nil, err2
		}
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	errUnknown := errors.New("unknown error in GetSnapDeclarationAssertion")
	logrus.Error(errUnknown)
	return nil, errUnknown
}

func (h *Handler) GetSnapRevisionAssertion(SHA3384Encoded string, rootStoreKey *rsa.PrivateKey, assertsDB *asserts.Database) (*asserts.SnapRevision, error) {
	revision, err := h.snaps.GetRevisionBySHA(SHA3384Encoded, true)
	if err == nil && revision != nil {
		snapEntry, err2 := h.snaps.GetSnapById(revision.SnapEntryID, true)
		logrus.Tracef("Got snap entry: %+v", snapEntry)

		if err2 == nil && snapEntry != nil {

			// TODO: we should get this somewhere sooner, like construction so we can MUST fail at the beginning of time
			storeAuthorityId := config.MustGetString(configkey.RootAuthority)

			// TODO: we can do better here
			assertion, err3 := asserts2.MakeSnapRevisionAssertion(storeAuthorityId, SHA3384Encoded, snapEntry.SnapStoreID, uint64(revision.Size), int(revision.ID), snapEntry.Account.AccountId,
				asserts.RSAPrivateKey(rootStoreKey).PublicKey().ID(), assertsDB)
			if err3 == nil && assertion != nil {
				return assertion, nil
			} else if err3 != nil {
				logrus.Error(err3)
				return nil, err3
			}
		} else if err2 != nil {
			logrus.Error(err2)
			return nil, err2
		}
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return nil, errors.New("unknown error encountered while trying to get snap revision assertion")
}

func (h *Handler) SnapDownload(snapFilename string) (*[]byte, error) {
	// TODO: make this part of construction
	obs := objectstore.NewObjectStore()

	bytes, err := obs.GetFileFromBucket("snaps", snapFilename)
	if err == nil && bytes != nil {
		return bytes, nil
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	logrus.Errorf("Error trying to get snap file %s for download", snapFilename)
	return nil, errors.New("unknown error encountered while trying to get snap for download")
}

func (h *Handler) SnapRefresh(actions *[]*requests.SnapActionJSON) (*responses.SnapActionResultList, error) {
	var actionResults []*responses.SnapActionResult
	for _, action := range *actions {
		snapEntry, err := h.snaps.GetSnap(action.Name, true)
		if err == nil && snapEntry != nil {
			// TODO: support other actions "refresh", etc.
			if action.Action == "download" {
				logrus.Infof("We know about this snap %s, its id is %s we we'll try to handle it.", snapEntry.Name, snapEntry.SnapStoreID)

				snapRevision, err2 := h.snaps.GetRevisionByChannel(action.Channel, action.Name)
				if err2 == nil && snapRevision != nil {
					storeSnap, err3 := snapEntry.ToStoreSnap(snapRevision)
					if err3 == nil && storeSnap != nil {
						actionResult := responses.SnapActionResult{
							Result:      "download",
							InstanceKey: "download-1",
							SnapID:      snapEntry.SnapStoreID,
							Name:        snapEntry.Name,
							Snap:        storeSnap,
						}

						actionResults = append(actionResults, &actionResult)
					}
					logrus.Errorf("unable to process action %s for snap %s: %s", action.Action, action.Name, err3)
				}
			} else if action.Action == "install" {
				logrus.Infof("We know about this snap %s, its id is %s we we'll try to handle it.", snapEntry.Name, snapEntry.SnapStoreID)
				snapRevision, err2 := h.snaps.GetRevisionByChannel(action.Channel, action.Name)
				if err2 == nil && snapRevision != nil {
					storeSnap, err3 := snapEntry.ToStoreSnap(snapRevision)
					if err3 == nil && storeSnap != nil {
						// TODO: this shouldn't be a fixed architecture
						storeSnap.Architectures = []string{"amd64"}
						storeSnap.Confinement = snapEntry.Confinement

						actionResult := responses.SnapActionResult{
							Result:      "install",
							InstanceKey: "install-1",
							SnapID:      snapEntry.SnapStoreID,
							Name:        snapEntry.Name,
							Snap:        storeSnap,
						}

						actionResults = append(actionResults, &actionResult)
					}
				}
			}
		} else if err != nil {
			logrus.Error(err)
		} else {
			logrus.Errorf("cannot process action %s for %s, snap unknown", action.Action, action.Name)
		}
	}

	actionResultList := responses.SnapActionResultList{
		Results:   actionResults,
		ErrorList: nil,
	}

	return &actionResultList, nil
}

func (h *Handler) FindSnap(name string) (*responses.SearchV2Results, error) {
	searchResult := responses.SearchV2Results{
		ErrorList: nil,
	}

	snapEntry, err := h.snaps.GetSnap(name, true)
	if err == nil && snapEntry != nil {
		results := func() []responses.StoreSearchResult {
			var results []responses.StoreSearchResult

			snapType := snap.TypeApp
			switch snapEntry.Type {
			case "os":
				snapType = snap.TypeOS
			case "snapd":
				snapType = snap.TypeSnapd
			case "base":
				snapType = snap.TypeBase
			case "gadget":
				snapType = snap.TypeGadget
			case "kernel":
				snapType = snap.TypeKernel
			}

			results = append(results, responses.StoreSearchResult{
				Revision: responses.StoreSearchChannelSnap{
					StoreSnap: responses.StoreSnap{
						Confinement: snapEntry.Confinement,
						CreatedAt:   snapEntry.CreatedAt.String(),
						Name:        snapEntry.Name,
						// TODO: need to fix this properly
						Revision:  1,
						SnapID:    snapEntry.SnapStoreID,
						Type:      snapType,
						Publisher: snap.StoreAccount{ID: snapEntry.Account.AccountId, Username: snapEntry.Account.Username, DisplayName: snapEntry.Account.DisplayName},
					},
				},
				Snap: responses.StoreSnap{
					Confinement: snapEntry.Confinement,
					CreatedAt:   snapEntry.CreatedAt.String(),
					Name:        snapEntry.Name,
					// TODO: need to fix this properly
					Revision:  1,
					SnapID:    snapEntry.SnapStoreID,
					Type:      snapType,
					Publisher: snap.StoreAccount{ID: snapEntry.Account.AccountId, Username: snapEntry.Account.Username, DisplayName: snapEntry.Account.DisplayName},
				},
				Name:   snapEntry.Name,
				SnapID: snapEntry.SnapStoreID,
			})

			return results
		}()

		searchResult.Results = results
		return &searchResult, nil
	} else if err != nil {
		logrus.Error(err)
		return nil, err
	}

	return nil, errors.New("unknown error encountered in FindSnap")
}

func (h *Handler) GetSnapNames() (*responses.CatalogResults, error) {
	snaps, err := h.snaps.GetSnaps()
	if err == nil && snaps != nil {
		catalogItems := responses.CatalogResults{
			Payload: responses.CatalogPayload{
				Items: []responses.CatalogItem{},
			},
		}

		for _, sn := range *snaps {
			catalogItems.Payload.Items = append(catalogItems.Payload.Items, responses.CatalogItem{
				Name: sn.Name,
				// TODO: implement version
				Version: "none provided",
				// TODO: implement summary
				Summary: "none provided",
				// TODO: implement aliases
				Aliases: nil,
				// TODO: implement apps
				Apps: nil,
				// TODO: implement title
				Title: "none provided",
			})
		}

		return &catalogItems, nil
	}

	if err != nil {
		return nil, err
	}

	return nil, errors.New("unknown error encountered")
}

func (h *Handler) GetSections() (*responses.SectionResults, error) {
	sections, err := h.snaps.GetSections()
	if err == nil && sections != nil {
		results := responses.SectionResults{
			Payload: responses.Payload{
				Sections: []responses.Section{
					{Name: "general"},
				},
			},
		}

		return &results, nil
	}

	return nil, errors.New("unknown error")
}

func createAccountAssertion(signingDB *assertstest.SigningDB, keyId string, accountId string, storeAccountUsername string) *asserts.Account {
	trustedAcctHeaders := map[string]interface{}{
		"validation": "certified",
		"timestamp":  "2015-11-20T15:04:00Z",
		"account-id": accountId,
	}

	trustedAcct := assertstest.NewAccount(signingDB, storeAccountUsername, trustedAcctHeaders, keyId)
	return trustedAcct
}

func getTrustedAccount(accountID string, signingDB *assertstest.SigningDB, displayName string) *asserts.Account {
	trustedAcctHeaders := map[string]interface{}{
		"validation": "verified",
		"timestamp":  "2015-11-20T15:04:00Z",
	}

	if displayName != "" {
		trustedAcctHeaders["display-name"] = displayName
	}

	trustedAcctHeaders["account-id"] = accountID
	trustedAcct := assertstest.NewAccount(signingDB, accountID, trustedAcctHeaders, "")

	return trustedAcct
}
