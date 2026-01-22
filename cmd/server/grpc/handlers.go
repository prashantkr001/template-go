package grpc

import (
	"context"

	"github.com/prashantkr001/template-go/cmd/server/grpc/proto/v1/pbitems"
	"github.com/prashantkr001/template-go/internal/item"
)

func (grp *GRPC) CreateItem(ctx context.Context, req *pbitems.CreateItemRequest) (*pbitems.Item, error) {
	createdItem, err := grp.apis.ItemCreateIfNotExists(ctx, item.Item{
		ID:   int(req.GetId()),
		Name: req.GetName(),
	})

	if err != nil {
		return nil, err
	}

	return &pbitems.Item{
		Id:   int64(createdItem.ID),
		Name: createdItem.Name,
	}, nil
}

func (grp *GRPC) ListItems(ctx context.Context, req *pbitems.ItemListRequest) (*pbitems.ItemListResponse, error) {
	list, err := grp.apis.ItemList(ctx, int(req.GetLimit()))
	if err != nil {
		return nil, err
	}

	ilist := make([]*pbitems.Item, 0, len(list))
	for i := range list {
		pbi := pbitems.Item{Id: int64(list[i].ID), Name: list[i].Name}
		ilist = append(ilist, &pbi)
	}

	return &pbitems.ItemListResponse{Items: ilist}, nil
}
