package chat

import (
	"fmt"
	"strconv"
	"time"

	csgf_client "github.com/Qwerty10291/csgf_bot/client"
	"github.com/Qwerty10291/csgf_bot/utils"
)

type mathGame struct {
	creator    string
	expression string
	answer     int
	bank       float32
}

func (g *mathGame) message() string {
	return fmt.Sprintf("%s, создал пример переведя на этот аккаунт %.2f. Пример:%s", g.creator, g.bank, g.expression)
}

type MathChatGame struct {
	client      *csgf_client.Client
	comission   float32
	currentGame *mathGame
	gamesQueue  []*mathGame
}

func NewMathChatGame(client *csgf_client.Client, comission float32) *MathChatGame {
	game := &MathChatGame{
		client:     client,
		comission:  comission,
		gamesQueue: []*mathGame{},
	}
	client.ChatUpdateHandler = game.messagesProcessor
	client.TransferEventHandler = game.transferHandler
	go game.adversion()
	return game
}

func (g *MathChatGame) messagesProcessor(msg *csgf_client.ChatEvent) {
	if g.currentGame != nil {
		answer, err := strconv.Atoi(msg.Message)
		if err == nil && answer == g.currentGame.answer {
			g.client.SendChatMessage(fmt.Sprintf("Победитель: %s", msg.Username))
			g.client.SendTransfer(msg.UserId, g.currentGame.bank)

			if len(g.gamesQueue) > 0 {
				game := g.gamesQueue[0]
				g.gamesQueue = g.gamesQueue[1:]
				g.startGame(game)
			} else {
				g.currentGame = nil
			}
		}
	}
}

func (g *MathChatGame) transferHandler(transfer *csgf_client.NotifyEventTransfer) {
	g.newGame(transfer.FromUser, transfer.Amount)
}

func (g *MathChatGame) newGame(creator string, bank float32) {
	bank = bank * (1 - g.comission)
	if bank < 1{
		bank = 1
	}

	expr, answer := utils.MathematicExpressionGenerator()
	game := &mathGame{
		creator:    creator,
		expression: expr,
		answer:     answer,
		bank:       bank,
	}

	if g.currentGame == nil {
		g.startGame(game)
	} else {
		g.gamesQueue = append(g.gamesQueue, game)
	}
}

func (g *MathChatGame) startGame(game *mathGame) {
	fmt.Println("new game", game.bank)
	g.currentGame = game
	g.client.SendChatMessage(game.message())
}

func (g *MathChatGame) adversion() {
	for {
		time.Sleep(time.Minute * 4)
		g.client.SendChatMessage("Вы можете воспользоваться функцией автоматического создания розыгрыша с примером, переведя на этот аккаунт любую сумму")
	}
} 

