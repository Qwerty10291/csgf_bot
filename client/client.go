package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)


var	messageSendInterval = time.Second

type csgfWebsocketAuthRequest struct {
	Params map[string]string `json:"params"`
	Id     int               `json:"id"`
}
type csgfWebsocketResult struct {
	Channel string                 `json:"channel"`
	Data    map[string]interface{} `json:"data"`
}
type csgfWebsocketResponse struct {
	Result csgfWebsocketResult `json:"result"`
}
type csgfBetResponse struct {
	Message struct {
		Status string `json:"status"`
		Text   string `json:"text"`
	} `json:"message"`
}

type clientInfo struct {
	balance float32
	token   string
	userId  int
}

type EventHandler func(channel string, event map[string]interface{})
type BalanceUpdateHandler func(BalanceEvent)

const (
	GameNew  GameUpdateReason = iota
	GameBet  GameUpdateReason = iota
	GameTime GameUpdateReason = iota
	GameEnd  GameUpdateReason = iota
)

type GameUpdateHandler func(*Game, GameUpdateReason)
type GameUpdateReason int

type ChatUpdateHandler func(*ChatEvent)
type TransferEventHandler func(*NotifyEventTransfer)

type ClientConfig struct {
	VkLogin              string
	VkPassword           string
	GameUpdateHandler    GameUpdateHandler
	ChatUpdateHandler    ChatUpdateHandler
	TransferEventHandler TransferEventHandler
}

type Client struct {
	ClientConfig
	httpClient    *http.Client
	websocket     *websocket.Conn
	Balance       float32
	UserId        int
	channelNotify string

	openedGames     map[int]*Game
	lastMessageTime time.Time
}

func NewClient(config ClientConfig) *Client {
	jar, _ := cookiejar.New(nil)
	client := http.Client{Jar: jar}

	return &Client{
		ClientConfig: config,
		httpClient:   &client,
		openedGames:  map[int]*Game{},
	}
}

func (c *Client) StartListener() {
	var resp csgfWebsocketResponse

	for {
		err := c.websocket.ReadJSON(&resp)
		if err != nil {
			fmt.Println("listener err", err)
		}
		switch resp.Result.Channel {
		case "new_game":
			newGameEvent, err := NewGameEventFromJson(resp.Result.Data)
			if err != nil {
				fmt.Println("new game event parse error", err)
				continue
			}
			c.processNewGameEvent(newGameEvent)
		case "end_game":
			event, err := EndGameEventFromJson(resp.Result.Data)
			if err != nil {
				fmt.Println("end game event parse error", err)
				continue
			}
			c.processEndGameEvent(event)
		case "new_bet":
			event, err := NewBetEventFromJson(resp.Result.Data)
			if err != nil {
				fmt.Println("new bet event parse error", err)
				continue
			}
			c.processNewBetEvent(event)
		case "chat_new":
			event, err := ChatEventFromJson(resp.Result.Data)
			if err != nil {
				fmt.Println("message parse error", err)
				continue
			}
			if c.ChatUpdateHandler != nil {
				c.ChatUpdateHandler(event)
			}
		case c.channelNotify:
			notifyType, notifyData, err := NotifyEventFromJson(resp.Result.Data)
			if err != nil {
				fmt.Println("notify event parse error", err)
				continue
			}
			switch notifyType {
			case NotifyTransfer:
				if c.TransferEventHandler != nil {
					transferEvent, _ := notifyData.(NotifyEventTransfer)
					c.TransferEventHandler(&transferEvent)
				}
			}
		}
	}
}

func (c *Client) Connect() error {
	if c.websocket != nil {
		return fmt.Errorf("already connected")
	}
	err := c.vkAuthorize()
	if err != nil {
		panic(err)
	}

	info, err := c.getClientInfo()
	if err != nil {
		return err
	}

	fmt.Println(info)
	conn, _, err := (&websocket.Dialer{Jar: c.httpClient.Jar}).Dial("wss://csgf.live/connection/websocket", nil)
	if err != nil {
		return err
	}

	conn.WriteJSON(csgfWebsocketAuthRequest{
		Params: map[string]string{"token": info.token},
		Id:     1,
	})
	var resp map[string]interface{}
	err = conn.ReadJSON(&resp)
	if err != nil {
		return err
	}
	if _, ok := resp["result"]; !ok {
		return fmt.Errorf("failed to auth in websocket")
	}
	fmt.Println(resp)

	err = conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"method":1,"params":{"channel":"test"},"id":2}
{"method":1,"params":{"channel":"new_bet"},"id":5}
{"method":1,"params":{"channel":"new_game"},"id":6}
{"method":1,"params":{"channel":"time_game"},"id":7}
{"method":1,"params":{"channel":"end_game"},"id":8}
{"method":1,"params":{"channel":"stats"},"id":9}
{"method":1,"params":{"channel":"balance#%d"},"id":10}
{"method":1,"params":{"channel":"chat_new"},"id":11}
{"method":1,"params":{"channel":"notify#%d"},"id":12}`, info.userId, info.userId)))
	if err != nil {
		return err
	}

	c.websocket = conn
	c.channelNotify = fmt.Sprintf("notify#%d", info.userId)
	c.Balance = info.balance
	c.UserId = info.userId
	return nil
}

func (c *Client) MakeBet(game *Game, summ float32) error {
	if summ > c.Balance {
		return fmt.Errorf("not enought balance")
	}
	resp, err := c.sendPostNultipart("https://csgf.live/bet", map[string]string{
		"gid": strconv.Itoa(game.Id),
		"sum": fmt.Sprintf("%.2f", summ)})
	if err != nil {
		return err
	}

	betResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	fmt.Println("bet resp", string(betResp))

	var data csgfBetResponse
	err = json.Unmarshal(betResp, &data)
	if err != nil {
		return err
	}

	if data.Message.Status != "success" {
		return fmt.Errorf("bet failed: {%s}", data.Message.Text)
	}
	game.BetNow += summ
	return nil
}

func (c *Client) SendChatMessage(msg string) {
	if interval := time.Now().Sub(c.lastMessageTime); interval < time.Second{
		time.Sleep(time.Second - interval + time.Millisecond * 100)
	}

	resp, err := c.sendPostNultipart("https://csgf.live/chat/send", map[string]string{"message": msg})
	if err != nil {
		panic(err)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))
}

func (c *Client) SendTransfer(userId int, summ float32) error {
	resp, err := c.sendPostNultipart("https://csgf.live/transfer", map[string]string{"id": strconv.Itoa(userId), "sum": fmt.Sprintf("%.2f", summ)})
	if err != nil {
		panic(err)
	}
	text := new(strings.Builder)
	io.Copy(text, resp.Body)
	fmt.Println("transfer response", text.String())
	return nil
}

func (c *Client) getClientInfo() (*clientInfo, error) {
	res, err := c.httpClient.Get("https://csgf.live/")

	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	html := string(data)
	tokenRegexp := regexp.MustCompile(`TOKEN = "(?P<token>.+?)"`)
	token := tokenRegexp.FindStringSubmatch(html)
	if token == nil {
		return nil, fmt.Errorf("cannot find token on page")
	}
	balanceRegexp := regexp.MustCompile(`<span class="balance">(.+)<\/span>`)
	balanceString := balanceRegexp.FindStringSubmatch(html)
	if balanceString == nil {
		return nil, fmt.Errorf("cannot find balance info")
	}
	balance, err := strconv.ParseFloat(balanceString[1], 32)
	if err != nil {
		return nil, err
	}

	userIdRegexp := regexp.MustCompile(`<a href="\/user\/(\d+)" class="avatar" title="Профиль">`)
	userIdData := userIdRegexp.FindStringSubmatch(html)
	if userIdData == nil {
		return nil, fmt.Errorf("failed to find user id")
	}
	userId, _ := strconv.Atoi(userIdData[1])

	return &clientInfo{
		balance: float32(balance),
		token:   token[1],
		userId:  userId,
	}, nil
}

func (c *Client) vkAuthorize() error {
	redirectResp, err := c.httpClient.Post("https://csgf.live/login", "multipart/form-data; boundary=-", nil)
	if err != nil {
		return err
	}
	var data map[string]string
	err = json.NewDecoder(redirectResp.Body).Decode(&data)
	if err != nil {
		return err
	}
	redirectUri := data["redirect"]
	vkResp, err := c.httpClient.Get(redirectUri)
	html := new(strings.Builder)
	_, err = io.Copy(html, vkResp.Body)
	if err != nil {
		return nil
	}

	fields := regexp.MustCompile(`<input type=\"hidden\" name=\"(.+)\" value=\"(.+)\"`).FindAllStringSubmatch(html.String(), -1)
	if fields == nil {
		return fmt.Errorf("cannot find fields in vk auth page")
	}

	form := url.Values{}
	for _, field := range fields {
		form.Add(field[1], field[2])
	}
	form.Add("email", c.VkLogin)
	form.Add("pass", c.VkPassword)
	form.Add("expire", "0")
	fmt.Println(form.Encode())

	request, _ := http.NewRequest("POST", "https://login.vk.com/?act=login&soft=1", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://oauth.vk.com")
	request.Header.Set("Referer", "https://oauth.vk.com")

	authResp, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	if authResp.Request.URL.String() != "https://csgf.live/" {
		return fmt.Errorf("vk auth failed")
	}
	return nil
}

func (c *Client) processNewGameEvent(event *NewGameEvent) {
	game, err := NewGame(event.GameId, event.Room)
	if err != nil {
		panic(err)
	}
	c.openedGames[game.Id] = game
	c.callGameUpdate(game, GameNew)
}

func (c *Client) processTimeEvent(event *TimeEvent) {
	if game, ok := c.openedGames[event.GameId]; ok {
		game.TimeNow = event.Time
		c.callGameUpdate(game, GameTime)
	}
}

func (c *Client) processNewBetEvent(event *NewBetEvent) {
	if game, ok := c.openedGames[event.GameId]; ok {

		game.Bank = event.CurrentBank
		if event.UserId != c.UserId {
			c.callGameUpdate(game, GameBet)
		}
	}
}

func (c *Client) processEndGameEvent(event *EndGameEvent) {
	if game, ok := c.openedGames[event.GameId]; ok {
		delete(c.openedGames, event.GameId)
		c.callGameUpdate(game, GameEnd)
	}
}

func (c *Client) callGameUpdate(game *Game, reason GameUpdateReason) {
	if c.GameUpdateHandler != nil {
		c.GameUpdateHandler(game, reason)
	}
}

func (c *Client) sendPostNultipart(url string, data map[string]string) (*http.Response, error) {
	body := new(bytes.Buffer)
	mp := multipart.NewWriter(body)
	mp.SetBoundary("----12232312412412")
	for key, value := range data {
		mp.WriteField(key, value)
	}
	return c.httpClient.Post(url, mp.FormDataContentType(), body)
}
