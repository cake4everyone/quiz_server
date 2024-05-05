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
	Twitch *twitchgo.Session
	WS     *websocket.Conn

	started time.Time
	Game    *Game

	userID string
}

type wsVoteMessage struct {
	Type     string `json:"type"`
	Username string `json:"username"`
}

// AllConnections is a map of a user id to connection for all currently active connections
var AllConnections = make(map[string]*Connection)

// New creates a new quiz connection for a new player. The user id is used to identify this client
// again for later requests.
//
// If there is already a connection with this user id New returns nil
func New(userID string) (c *Connection) {
	if _, found := GetConnection(userID); found {
		return nil
	}

	c = &Connection{
		userID:  userID,
		started: time.Now(),
	}

	AllConnections[userID] = c
	return c
}

// Used to re-obtain a connection, by passing in the user id. It also returns whether the
// connection was found.
func GetConnection(userID string) (*Connection, bool) {
	c, ok := AllConnections[userID]
	return c, ok
}

// Close is a graceful termination of the quiz connection to a player
func (c *Connection) Close() {
	delete(AllConnections, c.userID)

	if c.Twitch != nil {
		c.Twitch.Close()
		c.Twitch = nil
	}
	if c.WS != nil {
		c.WS.Close()
		c.WS = nil
	}
}

func (c *Connection) OnTwitchChannelMessage(t *twitchgo.Session, source *twitchgo.IRCUser, msg string) {
	if c.WS == nil || c.Game == nil {
		return
	}

	vote := MsgToVote(msg, c.Game)
	if vote == 0 {
		// ignoring non-valid votes
		return
	}
	if _, ok := c.Game.voteHistory[source.Nickname]; ok {
		// ignore users who already voted
		return
	}
	c.Game.voteHistory[source.Nickname] = true
	c.Game.ChatVoteCount[vote-1]++

	v := wsVoteMessage{
		Type:     "CHAT_VOTE",
		Username: source.Nickname,
	}

	err := c.WS.WriteJSON(v)
	if err != nil {
		log.Printf("Error writing chat vote to websocket: %v", err)
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

	if gameData.RoundDuration <= 0 {
		return fmt.Errorf("create game: round_duration must not be negative, got %ds", gameData.RoundDuration)
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
		connection:    c,
		Rounds:        rounds,
		RoundDuration: time.Duration(gameData.RoundDuration) * time.Second,
		Summary:       &GameSummary{},
		voteHistory:   make(map[string]bool),
	}

	return nil
}
