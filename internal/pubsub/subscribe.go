package pubsub

import (
	"encoding/json"
	"encoding/gob"
	"bytes"
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
	unmarshaller := func(data []byte) (T, error) {
		var target T
		err := json.Unmarshal(data, &target)
		if err != nil {
			fmt.Printf("SubscribeJSON, Unmarshal: %v\n", err)
			return target, err
		}
		return target, nil
	}
	subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)

	return nil
}

func SubscribeGob[T any](
    conn *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType SimpleQueueType, // an enum to represent "durable" or "transient"
    handler func(T) Acktype,
) error {
	unmarshaller := func(data []byte) (T, error) {
		buf := bytes.NewBuffer(data)
		dec := gob.NewDecoder(buf)
		var target T

		err := dec.Decode(&target)
		if err != nil {
			fmt.Printf("SubscribeGOB, Decode: %v\n", err)
			return target, err
		}
		return target, nil
	}
	subscribe(conn, exchange, queueName, key, queueType, handler, unmarshaller)

	return nil
}

func subscribe[T any](
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	simpleQueueType SimpleQueueType,
	handler func(T) Acktype,
	unmarshaller func([]byte) (T, error),
) error {
	chann, queue, err := DeclareAndBind(conn, exchange, queueName, key, simpleQueueType)
	if err != nil {
		fmt.Printf("SubscribeJSON, DeclareAndBind: %v\n", err)
		return err
	}

	err = chann.Qos(10, 0, false) 
	if err != nil {
		fmt.Printf("SubscribeJSON, DeclareAndBind: %v\n", err)
		return err
	}

	msgs, err := chann.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		fmt.Printf("Could no set QoS: %v\n", err)
		return err
	}

	go func() {
		defer chann.Close()
		for msg :=  range msgs {
			target, err := unmarshaller(msg.Body)
			if err != nil {
				fmt.Printf("subscribe, unmarshaller: %v\n", err)
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
