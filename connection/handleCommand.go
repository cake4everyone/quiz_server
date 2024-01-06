package connection

import (
	"encoding/json"
	"math/rand"
	"quiz_backend/quiz"
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
	CommandAUTH  TCPCommand = "AUTH"
	CommandEND   TCPCommand = "END"
	CommandERROR TCPCommand = "ERROR"
	CommandGAME  TCPCommand = "GAME"
	CommandINFO  TCPCommand = "INFO"
	CommandNEXT  TCPCommand = "NEXT"
	CommandSTATS TCPCommand = "STATS"
	CommandVOTE  TCPCommand = "VOTE"
)

func (c *Connection) handleAUTH(data string) {

}

func (c *Connection) handleEND(data string) {

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
		Rounds: rounds,
	}

	log.Printf("Created a new game for '%s' with %d rounds", c.conn.RemoteAddr().String(), len(c.Game.Rounds))

	c.sendNextRound()
}

func (c *Connection) handleNEXT(data string) {
	if c.Game.Current < len(c.Game.Rounds) {
		c.sendNextRound()
		return
	}
	c.ReplyERR("end reached")
}

func (c *Connection) handleVOTE(data string) {

}
