package main

import (
	"fmt"
	"log"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	fmt.Println("Peril game server connected to RabbitMQ!")

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not open a channel: %v", err)
	}

	playingState := new(routing.PlayingState)
	playingState.IsPaused = false
	pubsub.PublishJSON(
		channel, 
		routing.ExchangePerilDirect, 
		routing.PauseKey, 
		playingState,
	)	

	err = pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		pubsub.SimpleQueueDurable,
		handlerLog(),
	)
	if err != nil {
		log.Fatalf("could not subscribe to log: %v", err)
	}

	gamelogic.PrintServerHelp()

	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "pause":
			log.Print("Publishing a pause message...")
			err = pubsub.PublishJSON(
				channel, 
				routing.ExchangePerilDirect, 
				routing.PauseKey, 
				routing.PlayingState{IsPaused: true,},
			)	
			if err != nil {
				log.Printf("could not publish time: %v", err)
			}
		case "resume":
			log.Print("Publishing a resume game state message...")
			err = pubsub.PublishJSON(
				channel, 
				routing.ExchangePerilDirect, 
				routing.PauseKey, 
				routing.PlayingState{IsPaused: false,},
			)	
			if err != nil {
				log.Printf("could not publish resume game state: %v", err)
			}
		case "quit":
			log.Print("Exiting the game... Goodbye!")
			return
		default:
			log.Print("Command unknown :/")
		}
	}
}
