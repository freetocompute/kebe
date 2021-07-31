package store

import (
	"errors"

	"github.com/snapcore/snapd/snap"

	"github.com/sirupsen/logrus"

	"github.com/freetocompute/kebe/pkg/repositories"
	"github.com/freetocompute/kebe/pkg/store/responses"
)

type IStoreHandler interface {
	GetSections() (*responses.SectionResults, error)
	GetSnapNames() (*responses.CatalogResults, error)
	FindSnap(name string) (*responses.SearchV2Results, error)
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

func (h *Handler) FindSnap(name string) (*responses.SearchV2Results, error) {
	//var snapEntry models.SnapEntry
	//
	searchResult := responses.SearchV2Results{
		ErrorList: nil,
	}

	snapEntry, err := h.snaps.GetSnap(name, true)
	if err == nil && snapEntry != nil {

		//db := s.db.Preload(clause.Associations).Where(&models.SnapEntry{Name: name}).Find(&snapEntry)
		//if _, ok := database.CheckDBForErrorOrNoRows(db); ok {
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
		//
		//logrus.Infof("%+v", searchResult)
		//
		//c.Writer.Header().Set("Content-Type", "application/json")
		//bytes, _ := json.Marshal(&searchResult)
		//_, err2 := c.Writer.Write(bytes)
		//if err2 != nil {
		//	logrus.Error(err2)
		//	c.AbortWithStatus(http.StatusInternalServerError)
		//}
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
