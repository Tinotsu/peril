package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	const rabbitConnString = "amqp://guest:guest@localhost:5672/"

	conn, err := amqp.Dial(rabbitConnString)
	if err != nil {
		log.Fatalf("could not connect to RabbitMQ: %v", err)
	}
	defer conn.Close()
	fmt.Println("Peril game client connected to RabbitMQ!")

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not open a channel: %v", err)
	}

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("could not get username: %v", err)
	}

	gs := gamelogic.NewGameState(username)

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.ArmyMovesPrefix+"."+gs.GetUsername(), 
		routing.ArmyMovesPrefix+"."+"*",
		pubsub.SimpleQueueTransient,
		handlerMove(gs, channel),
	)
	if err != nil {
		log.Fatalf("could not move subsribe: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		routing.WarRecognitionsPrefix, 
		routing.WarRecognitionsPrefix+".*",
		pubsub.SimpleQueueDurable,
		handlerWar(gs, channel),
	)
	if err != nil {
		log.Fatalf("could not war subsribe: %v", err)
	}

	err = pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilDirect,
		routing.PauseKey+"."+gs.GetUsername(), 
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
		handlerPause(gs),
	)
	if err != nil {
		log.Fatalf("could not pause subscribe: %v", err)
	}

	for {
		input := gamelogic.GetInput()
		if len(input) == 0 {
			continue
		}
		switch input[0] {
		case "spawn":
			err = gs.CommandSpawn(input)
			if err != nil {
				log.Printf("could not make a command spawn: %v", err)
			}
		case "move":
			mv, err := gs.CommandMove(input)
			if err != nil {
				log.Printf("could not make a command move: %v", err)
			}
			err = pubsub.PublishJSON(
				channel,
				routing.ExchangePerilTopic,
				routing.ArmyMovesPrefix+"."+mv.Player.Username, 
				mv,
			)
			if err != nil {
				log.Printf("could not publish move: %v", err)
			}
			fmt.Printf("Moved %v units to %s\n", len(mv.Units), mv.ToLocation)
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			if len(input) < 2 {
				log.Print("A second argument is required\n")
			}
			number, err := strconv.Atoi(input[1])
			if err != nil {
				log.Printf("could not convert second argument to int: %v", err)
			}

			for i := 0; i < number; i++ {
				spamLog := new(routing.GameLog)
				spamLog.CurrentTime = time.Now()
				spamLog.Message = "spam"
				spamLog.Username = username
				err = pubsub.PublishGob(
					channel,
					routing.ExchangePerilTopic,
					routing.GameLogSlug+"."+username,
					spamLog,
				)
				if err != nil {
					log.Printf("could not publish log: %v", err)
				}
			}
		case "quit":
		gamelogic.PrintQuit()
		default:
		log.Print("Command unknown :/\n'help' to see the available commands\n")
		continue
		}
	}
}
