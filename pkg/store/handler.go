package store

import (
	"errors"

	"github.com/freetocompute/kebe/pkg/repositories"
	"github.com/freetocompute/kebe/pkg/store/responses"
)

type IStoreHandler interface {
	GetSections() (*responses.SectionResults, error)
	GetSnapNames() (*responses.CatalogResults, error)
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
