package pubsub

import (
	"encoding/json"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Acktype int

const (
	Ack Acktype = iota
	NackDiscard
	NackRequeue
)

func SubscribeJSON[T any](
    conn *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType SimpleQueueType, // an enum to represent "durable" or "transient"
    handler func(T) Acktype,
) error {
	chann, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		fmt.Printf("SubscribeJSON, DeclareAndBind: %v\n", err)
		return err
	}

	msgs, err := chann.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		fmt.Printf("SubscribeJSON, Consume: %v\n", err)
		return err
	}

	unmarshaller := func(data []byte) (T, error) {
		var target T
		err := json.Unmarshal(data, &target)
		return target, err
	}

	go func() {
		defer chann.Close()
		for msg :=  range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("SubscribeJSON, unmarshaller: %v\n", err)
				continue
			}
			ack := handler(target)
			switch ack {
			case Ack:
				msg.Ack(false)
			case NackRequeue:
				msg.Nack(false, true)
			case NackDiscard:
				msg.Nack(false, false)
			}
		}
	}()
	return nil
}
