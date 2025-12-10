package main

import (
	"fmt"
	"time"
	"log"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.Acktype {
	return func(rout routing.PlayingState) pubsub.Acktype {
		defer fmt.Print("> ")
		gs.HandlePause(rout)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, channel *amqp.Channel) func(gamelogic.ArmyMove) pubsub.Acktype {
	return func(move gamelogic.ArmyMove) pubsub.Acktype {
		defer fmt.Print("> ")
		out := gs.HandleMove(move)
		
		var ack pubsub.Acktype

		switch out {
		case gamelogic.MoveOutComeSafe:
			ack = pubsub.Ack
		case gamelogic.MoveOutcomeSamePlayer:
			ack = pubsub.Ack
		case gamelogic.MoveOutcomeMakeWar:
			recWar := new(gamelogic.RecognitionOfWar)
			recWar.Attacker = move.Player
			recWar.Defender = gs.GetPlayerSnap()

			err := pubsub.PublishJSON(
				channel,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.Player.Username, 
				recWar,
			)
			if err != nil {
				log.Printf("could not publish move: %v", err)
				ack = pubsub.NackRequeue
			}

			ack = pubsub.Ack

		default:
			ack = pubsub.NackDiscard
		}

		return ack
	}
}

func handlerWar(gs *gamelogic.GameState, channel *amqp.Channel) func(gamelogic.RecognitionOfWar) pubsub.Acktype {
	return func (rec gamelogic.RecognitionOfWar) pubsub.Acktype {
		defer fmt.Print("> ")
		out, winner, loser := gs.HandleWar(rec)

		var ack pubsub.Acktype
		msgLog := new(routing.GameLog)
		msgLog.CurrentTime = time.Now()
		msgLog.Username = gs.GetUsername()
		var msg string

		switch out {
		case gamelogic.WarOutcomeNotInvolved:
			ack = pubsub.NackRequeue
		case gamelogic.WarOutcomeNoUnits:
			ack = pubsub.NackDiscard
		case gamelogic.WarOutcomeOpponentWon:
			msg = fmt.Sprintf("%s won a war against %s", winner, loser)
			ack = pubsub.Ack
		case gamelogic.WarOutcomeYouWon:
			msg = fmt.Sprintf("%s won a war against %s", winner, loser)
			ack = pubsub.Ack
		case gamelogic.WarOutcomeDraw:
			msg = fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
			ack = pubsub.Ack
		default:
			ack = pubsub.NackDiscard
			log.Fatalf("could not declare war\n")
			msg = "ERROR wrong outcome a HandleWar\n"
		}

		msgLog.Message = msg
		err := PublishGameLog(channel, *msgLog)
		if err != nil {
		log.Fatalf("could not publish GameLog: %v", err)
			ack = pubsub.NackRequeue
		}

		return ack
	}
}

func PublishGameLog (channel *amqp.Channel, logMsg routing.GameLog) error {
	err := pubsub.PublishGob(
		channel,
		routing.ExchangePerilTopic,
		routing.GameLogSlug+"."+logMsg.Username, 
		logMsg,
	)
	if err != nil {
		return err
	}
	return nil
}
