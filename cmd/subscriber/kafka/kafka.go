// Package kafka is responsible for all subscription interfaces with Kafka
// Similar to the HTTP package, this should only have the "handlers" and none of the business logic
package kafka

import (
	"context"
	"sync"
	"time"

	"github.com/naughtygopher/errors"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/prashantkr001/template-go/internal/api"
	"github.com/prashantkr001/template-go/internal/pkg/kafka"
	"github.com/prashantkr001/template-go/internal/pkg/logger"
)

type Config struct {
	TopicItemCreate string
}
type Kafka struct {
	client *kafka.Kafka
	apiSvc *api.API
	// receivedFirstMessage is set to true if the subscriber successfully received
	// a message *ever*
	locker                 *sync.Mutex
	receivedFirstMessageAt *time.Time
	receivedLastMessageAt  *time.Time

	topicItemCreate string
}

func NewService(kfk *kafka.Kafka, apiSvc *api.API, cfg *Config) (*Kafka, error) {
	kf := &Kafka{
		client:          kfk,
		apiSvc:          apiSvc,
		locker:          &sync.Mutex{},
		topicItemCreate: cfg.TopicItemCreate,
	}

	return kf, nil
}

func (kfk *Kafka) Shutdown(ctx context.Context) error {
	if kfk == nil || kfk.client == nil {
		return nil
	}
	return kfk.client.Shutdown(ctx)
}

func (kfk *Kafka) ReceivedFirstMessageAt() *time.Time {
	var t *time.Time
	kfk.locker.Lock()
	defer kfk.locker.Unlock()

	if kfk.receivedFirstMessageAt != nil {
		tt := *kfk.receivedFirstMessageAt
		t = &tt
	}
	return t
}

func (kfk *Kafka) ReceivedLastMessageAt() *time.Time {
	var t *time.Time
	kfk.locker.Lock()
	defer kfk.locker.Unlock()

	if kfk.receivedLastMessageAt != nil {
		tt := *kfk.receivedLastMessageAt
		t = &tt
	}
	return t
}

func (kfk *Kafka) Subscribe(ctx context.Context) error {
	for {
		fetches := kfk.client.PollFetches(ctx)
		if errs := fetches.Errors(); len(errs) > 0 {
			// All errors are retried internally when fetching, but non-retriable errors are
			// returned from polls.
			return errors.Errorf("%+v", errs)
		}

		kfk.locker.Lock()
		now := time.Now()
		if kfk.receivedFirstMessageAt == nil {
			kfk.receivedFirstMessageAt = &now
		}
		kfk.receivedLastMessageAt = &now
		kfk.locker.Unlock()

		iter := fetches.RecordIter()
		recordCommits := make([]*kgo.Record, 0, fetches.NumRecords())
		for !iter.Done() {
			record := iter.Next()
			if record.Topic == kfk.topicItemCreate {
				kfk.client.HandleTopic(ctx, &recordCommits, record, kfk.ItemCreate)
			}
		}

		err := kfk.client.CommitRecords(ctx, recordCommits...)
		if err != nil {
			// the subscriber should not exit if there's a commit error. It should just log
			// and continue listening
			logger.ErrWithStacktrace(err)
		}
	}
}
