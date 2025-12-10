package main

import (
	"fmt"
	"log"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
)

func handlerLog() func(routing.GameLog) pubsub.Acktype {
	return func (gamelog routing.GameLog) pubsub.Acktype {
		defer fmt.Print("> ")
		err := gamelogic.WriteLog(gamelog)
		if err != nil {
			log.Fatalf("handlerLog, Writelog: %v", err)
			return pubsub.NackRequeue
		}
		return pubsub.Ack
	}
}
