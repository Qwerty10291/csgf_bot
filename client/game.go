package client

import (
	"math"
)

type Game struct {
	Id      int
	Bank    float32
	Room    string
	RoomId  int
	MaxBank float32
	MinBet  float32
	MaxBet  float32
	BetNow  float32
	TimeNow int
}

var MaxBankLimits = map[int]float32{
	1: 500,
	2: 50,
	3: 100,
	4: 2500,
	5: 10000,
	6: 50000,
}

var MaxBetLimits = map[int]float32{
	1: 50,
	2: 5,
	3: 50,
	4: 250,
	5: 1000,
	6: 5000,
}

var MinBetLimits = map[int]float32{
	1: 1,
	2: 0.1,
	3: 10,
	4: 10,
	5: 50,
	6: 250,
}

var RoomNames = map[int]string{
	1: "classic",
	2: "bich",
	3: "dual",
	4: "rich",
	5: "king",
	6: "epic",
}

var RoomTimes = map[int]int {
	1:15,
	2:10,
	3:1,
	4:20,
	5:25,
	6:30,
}

func NewGame(id int, roomId int) (*Game, error) {
	return &Game{
		Id:      id,
		Room:    RoomNames[roomId],
		RoomId:  roomId,
		MaxBank: MaxBankLimits[roomId],
		MinBet:  MinBetLimits[roomId],
		MaxBet:  MaxBetLimits[roomId],
		TimeNow: RoomTimes[roomId],
	}, nil
}

func (g *Game) GetCurrentPercent() float32 {
	return g.BetNow / g.Bank
}

func (g *Game) GetBetForPercent(percent float32) float32 {
	bet := float32(math.Floor(float64( (percent*g.Bank - g.BetNow) / (1 - percent)) * 100 ) / 100)
	if bet < 0{
		return 0
	}

	if bet < g.MinBet {
		return 0
	} else if bet > g.MaxBet {
		return 0
	} else if g.BetNow + bet > g.MaxBank{
		return 0
	} else {
		return bet
	}
}
