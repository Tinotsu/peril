package pubsub

import (
	"bytes"
	"encoding/gob"
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err := enc.Encode(val)
	if err != nil {
		return err
	}

	params := new(amqp.Publishing)
	params.ContentType = "application/gob"
	params.Body = buf.Bytes()

	ctx := context.Background()
	err = ch.PublishWithContext(ctx, exchange, key, false, false, *params)
	if err != nil {
		return err
	}

	return nil
}

