package item

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/naughtygopher/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pubMocker struct {
	pipe chan<- []byte
}

func (pMo *pubMocker) Publish(_ context.Context, item *Item) error {
	jbytes, err := json.Marshal(item)
	if err != nil {
		return errors.Wrap(err, "json marshal failed")
	}
	pMo.pipe <- jbytes
	return nil
}

func newPubMocker(pipe chan<- []byte) *pubMocker {
	return &pubMocker{
		// buffered channel
		pipe: pipe,
	}
}

type storeMocker struct {
	data map[int]Item
}

func (sMo *storeMocker) InsertItem(_ context.Context, item Item) (*Item, error) {
	sMo.data[item.ID] = item
	return &item, nil
}

func (sMo *storeMocker) Item(_ context.Context, id int) (*Item, error) {
	item, ok := sMo.data[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (sMo *storeMocker) ListItems(_ context.Context, limit int) ([]Item, error) {
	list := make([]Item, 0, limit)
	for idx := range sMo.data {
		list = append(list, sMo.data[idx])
		limit--
		if limit == 0 {
			break
		}
	}
	return list, nil
}

func newStoreMocker() *storeMocker {
	return &storeMocker{
		data: make(map[int]Item),
	}
}

// TestInsertItem ensures the business logic is in place
/*
It tests the following scenarios
1. Name mutation.
2. ID validation.
3. Is it persisting.
4. Is it pushing to publisher.
*/
func TestInsertItem(t *testing.T) {
	requirer := require.New(t)
	asserter := assert.New(t)
	pipe := make(chan []byte, 128)
	pmo := newPubMocker(pipe)
	smo := newStoreMocker()
	svc, err := NewService(smo, pmo)
	requirer.NoError(err)
	ctx := t.Context()

	item := Item{
		ID:   123,
		Name: "Bottle",
	}
	insertedItem, err := svc.CreateIfNotExist(ctx, item)
	requirer.NoError(err)

	t.Run("testing the special number suffix business logic", func(_ *testing.T) {
		parts := strings.Split(insertedItem.Name, "-")
		if asserter.Len(parts, 2) {
			_, err = strconv.Atoi(parts[1])
			requirer.NoError(err)
		}
	})

	t.Run("check if item was persisted in the storage", func(_ *testing.T) {
		storedItem := smo.data[insertedItem.ID]
		parts := strings.Split(storedItem.Name, "-")
		if asserter.Len(parts, 2) {
			// while testing equality, the random generated number suffix is removed
			storedItem.Name = parts[0]
		}
		asserter.Equal(item, storedItem)
	})

	t.Run("check if item was pushed to the publisher", func(_ *testing.T) {
		pBytes := <-pipe
		pubItem := Item{}
		err = json.Unmarshal(pBytes, &pubItem)
		requirer.NoError(err)
		parts := strings.Split(pubItem.Name, "-")
		if asserter.Len(parts, 2) {
			// while testing equality, the random generated number suffix is removed
			pubItem.Name = parts[0]
		}
		asserter.Equal(item, pubItem)
	})

	t.Run("testing if the ID validations are in place", func(_ *testing.T) {
		item.ID = 0
		_, err = svc.Create(ctx, item)
		requirer.ErrorIs(err, ErrInvalidID)
		item.ID = -1
		_, err = svc.Create(ctx, item)
		requirer.ErrorIs(err, ErrInvalidID)
	})
}
