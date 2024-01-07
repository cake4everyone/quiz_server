package connection

import (
	"encoding/json"
	"math/rand"
	"net"
	"quiz_backend/quiz"

	"github.com/kesuaheli/twitchgo"
)

type roundData struct {
	Question string   `json:"question"`
	Answers  []string `json:"answers"`
	Current  int      `json:"current_round"`
	Max      int      `json:"max_round"`
	Category string   `json:"category"`
}

type TCPCommand string

const (
	CommandAUTH        TCPCommand = "AUTH"
	CommandEND         TCPCommand = "END"
	CommandERROR       TCPCommand = "ERROR"
	CommandGAME        TCPCommand = "GAME"
	CommandINFO        TCPCommand = "INFO"
	CommandNEXT        TCPCommand = "NEXT"
	CommandSTATS       TCPCommand = "STATS"
	CommandSTATSGAME   TCPCommand = "STATS game"
	CommandSTATSGLOBAL TCPCommand = "STATS global"
	CommandSTATSROUND  TCPCommand = "STATS round"
	CommandVOTE        TCPCommand = "VOTE"
)

func (c *Connection) handleAUTH(data string) {
	twitch := twitchgo.New("", data)
	twitch.OnGlobalUserState(func(t *twitchgo.Twitch, userTags twitchgo.MessageTags) {
		t.JoinChannel(userTags.DisplayName)
		c.Reply(CommandINFO, []byte("logged in as "+userTags.DisplayName))
	})
	err := twitch.Connect()
	if err != nil {
		if opErr, ok := err.(*net.OpError); ok {
			err = opErr.Unwrap()
		}
		c.ReplyERR(err.Error() + ": Is your token correct and valid?")
		return
	}
	twitch.OnChannelMessage(func(t *twitchgo.Twitch, channel string, source *twitchgo.User, msg string) {
		log.Printf("[Twitch] <%s %s> %s", channel, source.Nickname, msg)
		t.SendMessage(channel, "test")
	})
}

func (c *Connection) handleEND(data string) {
	switch data {
	case "round":
		if c.Game == nil {
			c.ReplyERR("no game running")
			return
		}
		c.sendStatsRound()
		if c.Game.Current >= len(c.Game.Rounds) {
			c.handleEND("game")
			return
		}
		c.Game.Current++
		c.sendCurrentRound()
	case "game":
		if c.Game == nil {
			c.ReplyERR("no game running")
			return
		}
		c.sendStatsGame()
		c.Game = nil
	}
}

// temporary data
func (c *Connection) handleGAME(data string) {
	var gameData struct {
		Categories    map[string]int `json:"categories"`
		RoundDuration int            `json:"round_duration"`
	}
	err := json.Unmarshal([]byte(data), &gameData)
	if err != nil {
		log.Printf("Failed to unmarshal game data: %v", err)
		c.ReplyERR("Bad Request")
		return
	}
	var rounds []*quiz.Round
	for name, n := range gameData.Categories {
		cat := quiz.Categories.GetCategoryByName(name)
		if cat.Title == "" {
			log.Printf("Failed to get category %s: unknown category", name)
			c.ReplyERRf("Unknown Category: %s", name)
			return
		}
		rounds = append(rounds, cat.GetRounds(n)...)
	}
	max := len(rounds)
	if max == 0 {
		log.Println("Failed to create game: too few questions")
		c.ReplyERR("Bad Request")
		return
	}

	rand.Shuffle(max, func(i, j int) {
		rounds[i], rounds[j] = rounds[j], rounds[i]
	})

	for i, r := range rounds {
		r.Current = i + 1
		r.Max = max
	}

	c.Game = &quiz.Game{
		Rounds:  rounds,
		Current: 1,
	}

	log.Printf("Created a new game for '%s' with %d rounds", c.conn.RemoteAddr().String(), len(c.Game.Rounds))

	c.sendCurrentRound()
}

func (c *Connection) handleNEXT(data string) {
}

func (c *Connection) handleVOTE(data string) {
}
