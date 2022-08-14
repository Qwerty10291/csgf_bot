package client

import (
	"fmt"
	"regexp"
	"strconv"
)

type NewGameEvent struct {
	GameId int
	Room   int
}

func NewGameEventFromJson(data map[string]interface{}) (*NewGameEvent, error) {
	eventData, ok := (data["data"]).(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data filed not found")
	}

	room, err := strconv.Atoi(eventData["room"].(string))
	if err != nil {
		return nil, fmt.Errorf("failed to parse room field")
	}

	gameId, err := strconv.Atoi(regexp.MustCompile(`game_(\d+)`).FindStringSubmatch(eventData["blade"].(string))[1])
	if err != nil {
		return nil, err
	}
	return &NewGameEvent{
		GameId: gameId,
		Room:   room,
	}, nil
}

type EndGameEvent struct {
	GameId int
}

func EndGameEventFromJson(data map[string]interface{}) (*EndGameEvent, error) {
	gameId, err := strconv.Atoi((data["data"]).(map[string]interface{})["game"].(string))
	if err != nil {
		return nil, fmt.Errorf("cannot parse gameId")
	}
	return &EndGameEvent{gameId}, nil
}

type NewBetEvent struct {
	CurrentBank float32
	GameId      int
	UserId      int
	Summ 		float32
}

func NewBetEventFromJson(data map[string]interface{}) (*NewBetEvent, error) {
	eventData, ok := data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("event data not found")
	}
	currentBank, err := strconv.ParseFloat(eventData["bank"].(string), 32)
	if err != nil {
		return nil, fmt.Errorf("cannot convert bank field (%s) to float", eventData["bank"].(string))
	}
	gameId, err := strconv.Atoi(eventData["game"].(string))
	if err != nil {
		return nil, fmt.Errorf("cannot parse gameId %s", eventData["game"].(string))
	}
	html := eventData["blade"].(string)
	
	usersRegex := regexp.MustCompile(`<a href="\/user\/(\d+)">`)
	usersData := usersRegex.FindStringSubmatch(html)
	if usersData == nil {
		return nil, fmt.Errorf("cannot find user id")
	}
	userId, err := strconv.Atoi(usersData[1])
	if err != nil {
		return nil, fmt.Errorf("cannot parse user id")
	}
	summRegexp := regexp.MustCompile(`<span class="sum">(.+) <`)
	summData := summRegexp.FindStringSubmatch(html)
	if summData == nil{
		return nil, fmt.Errorf("cannot find bet")
	}
	bet, err := strconv.ParseFloat(summData[1], 32)
	if err != nil{
		return nil, fmt.Errorf("cannot parse bet")
	}
	return &NewBetEvent{
		CurrentBank: float32(currentBank),
		GameId:      gameId,
		UserId:      userId,
		Summ: float32(bet),
	}, nil
}

type BalanceEvent struct {
	Balance float32
}

func BalanceEventFromJson(data map[string]interface{}) (*BalanceEvent, error) {
	if balance, ok := data["data"].(map[string]interface{})["balance"].(float32); ok {
		return &BalanceEvent{Balance: balance}, nil
	}
	return nil, fmt.Errorf("balance field not found")
}

type TimeEvent struct {
	Time   int
	Room   int
	GameId int
}

func TimeEventFromJson(data map[string]interface{}) (*TimeEvent, error) {
	eventData, ok := data["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("data field not found")
	}
	gameId, ok := eventData["game"].(int)
	if !ok {
		return nil, fmt.Errorf("game field not found")
	}
	room, ok := eventData["room"].(int)
	if !ok {
		return nil, fmt.Errorf("room field not found")
	}
	time, ok := eventData["time"].(int)
	if !ok {
		return nil, fmt.Errorf("time field not found")
	}
	return &TimeEvent{
		Time:   time,
		Room:   room,
		GameId: gameId,
	}, nil
}

type ChatEvent struct{
	Message string
	UserId int
	Username string
}

func ChatEventFromJson(data map[string]interface{}) (*ChatEvent, error) {
	html, ok := data["data"].(map[string]interface{})["blade"].(string)
	if !ok {
		return nil, fmt.Errorf("blade field not found")
	}
	messageData := regexp.MustCompile(`<span class="text2">(.+)<\/span>`).FindStringSubmatch(html)
	if messageData == nil{
		return nil, fmt.Errorf("cannot parse message")
	}

	userIdData := regexp.MustCompile(`data-user="(\d+)"`).FindStringSubmatch(html)
	if userIdData == nil{
		return nil, fmt.Errorf("cannot find user id")
	}
	userId, _ := strconv.Atoi(userIdData[1])

	usernameData := regexp.MustCompile(`data-text="(.+?)"`).FindStringSubmatch(html)
	if usernameData == nil{
		return nil, fmt.Errorf("cannot find user id")
	}
	return &ChatEvent{
		Message: messageData[1],
		UserId: userId,
		Username: usernameData[1],
	}, nil
}

type NotifyEventType int
const (
	NotifyUnknown NotifyEventType = iota
	NotifyTransfer NotifyEventType = iota
)

type NotifyEventTransfer struct{
	Amount float32
	FromUser string
}

func NotifyEventFromJson(data map[string]interface{}) (NotifyEventType, interface{}, error) {
	eventText, ok := data["data"].(map[string]interface{})["message"].(map[string]interface{})["text"].(string)
	if !ok{
		return NotifyUnknown, nil, fmt.Errorf("cannot find event text")
	}
	
	transferRegexp := regexp.MustCompile(`Переведено (.+)<br>от (.+)`)
	transferData := transferRegexp.FindStringSubmatch(eventText)
	if transferData != nil{
		amount, _ := strconv.ParseFloat(transferData[1], 32)
		return NotifyTransfer, NotifyEventTransfer{float32(amount), transferData[2]}, nil
	}

	return NotifyUnknown, nil, nil
}