// Package api maintains all the APIs exposed by this application
/*
It's beneficial to prefix the API with the respective module, so that it's easier for devs to go through all APIs of a given module. It would also help avoid conflict between same names across packages.

e.g. users.Create(), and items.Create(), both are named Create and is
an appropriate name within the package. But in the API package, it's better of to name them
UserCreate & ItemCreate.
*/
package api

import (
	"context"

	"github.com/prashantkr001/template-go/internal/item"
)

type itemService interface {
	CreateIfNotExist(ctx context.Context, newItem item.Item) (*item.Item, error)
	List(ctx context.Context, limit int) ([]item.Item, error)
}

// API struct holds all the initialized service structs of respective modules, which has
// its API exposed.
type API struct {
	itemService itemService
}

func NewService(itSvc itemService) *API {
	return &API{itemService: itSvc}
}
