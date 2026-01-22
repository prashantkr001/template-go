package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/prashantkr001/template-go/internal/item"
	"github.com/prashantkr001/template-go/internal/pkg/logger"
)

func (kfk *Kafka) ItemCreate(ctx context.Context, payload []byte) error {
	createItem := new(item.Item)
	err := json.Unmarshal(payload, createItem)
	if err != nil {
		// log the error and move on, if `nack`-ed, app will receive the same message, and the error.
		// Ending up in an infinite loop or Kafka backing off from delivering messages to the consumer
		// group
		logger.ErrWithStacktrace(fmt.Errorf("%q %w", string(payload), err))
		return nil
	}

	_, err = kfk.apiSvc.ItemCreateIfNotExists(ctx, *createItem)
	// we could use errors.Is and make further checks to see if the error can be fixed upon retry.
	// if it's an unrecoverable error, then it's better to log and return nil from here so the
	// message will be committed/ack-ed.
	if errors.Is(err, item.ErrDuplicateItem) {
		logger.Info(fmt.Sprintf("item with ID %d already exists", createItem.ID))
		return nil
	}

	return err
}
