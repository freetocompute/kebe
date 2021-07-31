package store

import (
	"errors"

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
			if action.Action == "download" {
				logrus.Infof("We know about this snap %s, its id is %s we we'll try to handle it.", snapEntry.Name, snapEntry.SnapStoreID)

				snapRevision, err2 := h.snaps.GetRevisionByChannel(action.Channel, action.Name) // //s.getRevision(action.Channel, action.Name)
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
				snapRevision, err2 := h.snaps.GetRevisionByChannel(action.Channel, action.Name) // //s.getRevision(action.Channel, action.Name)
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
