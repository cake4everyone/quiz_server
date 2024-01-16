package quiz

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kesuaheli/twitchgo"
)

// Connection represents a connection to a logged in player
type Connection struct {
	Twitch *twitchgo.Twitch
	WS     *websocket.Conn

	started time.Time
	Game    *Game

	token string
}

type wsVoteMessage struct {
	Type     string `json:"type"`
	Username string `json:"username"`
	Vote     int    `json:"vote"`
	Total    [4]int `json:"chat_vote_count"`
}

// AllConnections is a map of a token to connection for all currently active connections
var AllConnections = make(map[string]*Connection)

// New creates a new quiz connection for a new player. The token is used to identify this client
// again for later requests.
//
// If there is already a connection with this token New returns nil
func New(token string) (c *Connection) {
	if _, found := GetConnectionByToken(token); found {
		return nil
	}

	c = &Connection{
		token:   token,
		started: time.Now(),
	}

	AllConnections[token] = c
	return c
}

// Used to re-obtain a connection, by passing in the connection token. It also returns whether the
// connection was found.
func GetConnectionByToken(token string) (*Connection, bool) {
	c, ok := AllConnections[token]
	return c, ok
}

// Close is a graceful termination of the quiz connection to a player
func (c *Connection) Close() {
	delete(AllConnections, c.token)

	if c.Twitch != nil {
		c.Twitch.Close()
		c.Twitch = nil
	}
	if c.WS != nil {
		c.WS.Close()
		c.WS = nil
	}
}

func (c *Connection) OnTwitchChannelMessage(t *twitchgo.Twitch, channel string, source *twitchgo.User, msg string) {
	if c.WS == nil || c.Game == nil {
		return
	}

	v := wsVoteMessage{
		Type:     "CHAT_VOTE",
		Username: source.Nickname,
		Vote:     1,
		Total:    c.Game.ChatVoteCount,
	}

	switch msg {
	case "1", "a", "A":
		c.Game.ChatVoteCount[0]++
		v.Vote = 1
		c.WS.WriteJSON(v)
	case "2", "b", "B":
		c.Game.ChatVoteCount[1]++
		v.Vote = 2
		c.WS.WriteJSON(v)
	case "3", "c", "C":
		c.Game.ChatVoteCount[2]++
		v.Vote = 3
		c.WS.WriteJSON(v)
	case "4", "d", "D":
		c.Game.ChatVoteCount[3]++
		v.Vote = 4
		c.WS.WriteJSON(v)
	}
}

func (c *Connection) NewGame(data []byte) error {
	var gameData struct {
		Categories    map[string]int `json:"categories"`
		RoundDuration int            `json:"round_duration"`
	}
	err := json.Unmarshal(data, &gameData)
	if err != nil {
		return fmt.Errorf("create game: %v", err)
	}

	var rounds []*Round
	for name, n := range gameData.Categories {
		cat := Categories.GetCategoryByName(name)
		if cat.Title == "" {
			return fmt.Errorf("create game: unknown category '%s'", name)
		}
		rounds = append(rounds, cat.GetRounds(n)...)
	}
	max := len(rounds)
	if max == 0 {
		return fmt.Errorf("create game: too few question")
	}

	rand.Shuffle(max, func(i, j int) {
		rounds[i], rounds[j] = rounds[j], rounds[i]
	})

	for i, r := range rounds {
		r.Current = i + 1
		r.Max = max
	}

	c.Game = &Game{
		Rounds:  rounds,
		Current: 1,
	}

	return nil
}
