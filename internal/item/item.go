// Package item is responsible for implementing all features required for handling Item
package item

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/naughtygopher/errors"

	"github.com/prashantkr001/template-go/internal/pkg/logger"
)

var (
	ErrInvalidID = errors.Validation("item ID should be >= 0")

	ErrNotFound      = errors.NotFound("Item not found")
	ErrDuplicateItem = errors.Duplicate("Item with the same ID already exists")
)

type Item struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func (it *Item) Validate() error {
	if it.ID <= 0 {
		return errors.Wrap(ErrInvalidID, fmt.Sprintf("'%d'", it.ID))
	}
	return nil
}

// Service struct holds all the dependencies required, as interfaces. e.g. persistent store interface,
// cache interface etc.
// And all its usecases as methods(with pointer receiver) of this struct.
type Service struct {
	persistentStore persistentStore
	publisher       publisher
}

// NewService accepts any external dependencies required for the campaign service.
// e.g. DB driver.
func NewService(storage persistentStore, pub publisher) (*Service, error) {
	return &Service{
		persistentStore: storage,
		publisher:       pub,
	}, nil
}

func (svc *Service) Create(ctx context.Context, item Item) (*Item, error) {
	// do validations of values in item here or other business logic
	err := item.Validate()
	if err != nil {
		return nil, err
	}

	// my *business logic* requires Item name to be suffixed with a random number
	item.Name = fmt.Sprintf("%s-%d", item.Name, rand.Int()) //nolint:gosec // G404: Non-crypto usage, just for naming uniqueness

	newItem, err := svc.persistentStore.InsertItem(ctx, item)
	if err != nil {
		return nil, err
	}

	// publish the newly created items, for all dependencies to consume
	go func() {
		// since this is asynchronous, maybe implement some retry logic if required
		const publishTimeout = time.Second * 3
		gctx, cancel := context.WithTimeout(context.Background(), publishTimeout)
		defer cancel()

		perr := svc.publisher.Publish(gctx, newItem)
		if perr != nil {
			logger.ErrWithStacktrace(perr)
			return
		}
		logger.Info(fmt.Sprintf("published to kafka: %v", newItem))
	}()

	// if you have a cache, set the cache here. Similar to publish events
	// you might want to implement retry mechanism for writing to cache.
	// Also, would be a good idea to do it asynchronously, since the API
	// need not be (depends on how critical you think this is) blocked.
	// Remember to log the erorr if you're setting the cache asynchronously

	return newItem, nil
}

func (svc *Service) CreateIfNotExist(ctx context.Context, item Item) (*Item, error) {
	_, err := svc.persistentStore.Item(ctx, item.ID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	// expected ErrNotFound
	if err == nil {
		return nil, errors.Wrapf(ErrDuplicateItem, ": %d", item.ID)
	}

	return svc.Create(ctx, item)
}

func (svc *Service) List(ctx context.Context, limit int) ([]Item, error) {
	list, err := svc.persistentStore.ListItems(ctx, limit)
	if err != nil {
		return nil, err
	}

	return list, nil
}
