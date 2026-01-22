package item

import (
	"context"
	"encoding/json"

	"github.com/naughtygopher/errors"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/prashantkr001/template-go/internal/pkg/kafka"
)

type publisher interface {
	Publish(ctx context.Context, item *Item) error
}

type kafkaItemPublisher struct {
	cli              *kafka.Kafka
	afterCreateTopic string
}

func NewKafkaItemPublisher(
	kcli *kafka.Kafka,
	pubTopic string,
) (*kafkaItemPublisher, error) { //nolint:revive // it is ok to return unexported type in this case, ensures controlled access
	return &kafkaItemPublisher{
		cli:              kcli,
		afterCreateTopic: pubTopic,
	}, nil
}

func (kip *kafkaItemPublisher) Publish(ctx context.Context, item *Item) error {
	jbytes, err := json.Marshal(item)
	if err != nil {
		return errors.Wrap(err, "json marshal failed")
	}

	err = kip.cli.ProduceSync(ctx, &kgo.Record{Value: jbytes, Topic: kip.afterCreateTopic})
	if err != nil {
		return errors.Wrap(err, "kafka produce sync failed")
	}

	return nil
}
