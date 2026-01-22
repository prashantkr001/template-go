package item

import (
	"context"

	"github.com/naughtygopher/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type persistentStore interface {
	InsertItem(ctx context.Context, item Item) (*Item, error)
	ListItems(ctx context.Context, limit int) ([]Item, error)
	Item(ctx context.Context, id int) (*Item, error)
}

type mongoItemStore struct {
	mongoDriver    *mongo.Database
	itemCollection *mongo.Collection
}

func NewMongoPersistentStore(client *mongo.Database) (*mongoItemStore, error) { //nolint:revive // it is ok to return unexported type in this case, ensures controlled access
	istore := &mongoItemStore{
		mongoDriver:    client,
		itemCollection: client.Collection("items"),
	}
	return istore, nil
}

func (istore *mongoItemStore) InsertItem(ctx context.Context, item Item) (*Item, error) {
	_, err := istore.itemCollection.InsertOne(ctx, item)
	if err != nil {
		return nil, errors.Wrap(err, "could not save the item")
	}

	return &item, nil
}

func (istore *mongoItemStore) Item(ctx context.Context, id int) (*Item, error) {
	result := istore.itemCollection.FindOne(ctx, bson.M{"id": bson.M{"$eq": id}})
	item := new(Item)
	err := result.Decode(item)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, errors.Wrap(err, "failed getting item")
	}

	return item, nil
}

func (istore *mongoItemStore) ListItems(ctx context.Context, limit int) ([]Item, error) {
	result, err := istore.itemCollection.Find(ctx, bson.M{}, options.Find().SetLimit(int64(limit)))
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch items")
	}

	list := make([]Item, 0, limit)
	err = result.All(ctx, &list)
	if err != nil {
		return nil, errors.Wrap(err, "could not fetch items")
	}

	return list, nil
}
