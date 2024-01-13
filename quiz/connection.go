package quiz

import (
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
	log.Printf("[Twitch] <%s %s> %s", channel, source.Nickname, msg)
}
