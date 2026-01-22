package api

import (
	"context"

	"github.com/prashantkr001/template-go/internal/item"
)

func (ap *API) ItemCreateIfNotExists(ctx context.Context, newItem item.Item) (*item.Item, error) {
	createdItem, err := ap.itemService.CreateIfNotExist(ctx, newItem)
	if err != nil {
		return nil, err
	}
	return createdItem, nil
}

func (ap *API) ItemList(ctx context.Context, limit int) ([]item.Item, error) {
	list, err := ap.itemService.List(ctx, limit)
	if err != nil {
		return nil, err
	}
	return list, nil
}
