package connection

import (
	"encoding/json"
	"fmt"
	"quiz_backend/quiz"
)

func (c Connection) Reply(cmd TCPCommand, data []byte) {
	if len(data) > 0 {
		data = append([]byte(cmd+" "), data...)
	} else {
		data = []byte(cmd)
	}
	data = append(data, '\n')
	c.conn.Write(data)
}

func (c Connection) ReplyERR(msg string) {
	c.Reply(CommandERROR, []byte("{\"error\":\""+msg+"\"}"))
}

func (c Connection) ReplyERRf(format string, a ...any) {
	c.ReplyERR(fmt.Sprintf(format, a...))
}

func (c *Connection) sendCurrentRound() {
	round := *c.Game.Rounds[c.Game.Current-1]
	round.Correct = 0
	b, err := json.Marshal(round)
	if err != nil {
		log.Printf("Failed to marshal round data: %v", err)
		c.ReplyERR("Internal Server Error")
		return
	}
	c.Reply(CommandNEXT, b)
}

func (c Connection) sendStatsRound() {
	round := quiz.RoundSummary{
		Round:          c.Game.Rounds[c.Game.Current-1],
		StreamerPoints: c.Game.Summary.StreamerPoints,
		StreamerVote:   c.Game.StreamerVote,
		ChatPoints:     c.Game.Summary.ChatPoints,
		ChatVote:       c.Game.ChatVote,
		ChatVoteCount:  c.Game.ChatVoteCount,
	}

	b, err := json.Marshal(round)
	if err != nil {
		log.Printf("Failed to marshal round data: %v", err)
		c.ReplyERR("Internal Server Error")
		return
	}
	c.Reply(CommandSTATSROUND, b)
}

func (c Connection) sendStatsGame() {
	b, err := json.Marshal(c.Game.Summary)
	if err != nil {
		log.Printf("Failed to marshal round data: %v", err)
		c.ReplyERR("Internal Server Error")
		return
	}
	c.Reply(CommandSTATSGAME, b)
}
