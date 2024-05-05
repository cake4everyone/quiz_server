package quiz

import (
	logger "log"
	"math/rand"
	"time"
)

type Game struct {
	connection *Connection

	Current       int
	Rounds        []*Round
	RoundDuration time.Duration
	RoundTimer    *time.Timer

	StreamerVote  int             `json:"streamer_vote"`
	ChatVote      int             `json:"chat_vote"`
	ChatVoteCount [4]int          `json:"chat_vote_count"`
	voteHistory   map[string]bool `json:"-"`
	Summary       GameSummary
}

type GameSummary struct {
	StreamerPoints int `json:"streamer_points"`
	StreamerWon    int `json:"streamer_won"`
	ChatPoints     int `json:"chat_points"`
	ChatWon        int `json:"chat_won"`
}

type Category struct {
	Title       string      `json:"-"`
	Description string      `json:"description"`
	Pool        []*Question `json:"pool"`
}

type Question struct {
	Question string   `json:"question"`
	Correct  []string `json:"correct"`
	Wrong    []string `json:"wrong"`
}
type Round struct {
	Question string   `json:"question"`
	Answers  []string `json:"answers"`
	Correct  int      `json:"correct,omitempty"`
	Current  int      `json:"current_round"`
	Max      int      `json:"max_round"`
	Category string   `json:"category"`
}

type RoundSummary struct {
	*Round
	StreamerPoints int    `json:"streamer_points"`
	StreamerVote   int    `json:"streamer_vote"`
	ChatPoints     int    `json:"chat_points"`
	ChatVote       int    `json:"chat_vote"`
	ChatVoteCount  [4]int `json:"chat_vote_count"`
}

type categoriesSlice []Category

// Categories is the main list of all Categories and Categories
var Categories categoriesSlice

var log = logger.New(logger.Writer(), "[WEB] ", logger.LstdFlags|logger.Lmsgprefix)

func (cs categoriesSlice) GetCategoryByName(name string) Category {
	for _, c := range cs {
		if c.Title == name {
			return c
		}
	}
	return Category{}
}

func (g Game) GetRoundSummary() RoundSummary {
	sum := RoundSummary{
		StreamerPoints: g.Summary.StreamerPoints,
		StreamerVote:   g.StreamerVote,
		ChatPoints:     g.Summary.ChatPoints,
		ChatVote:       g.ChatVote,
		ChatVoteCount:  g.ChatVoteCount,
	}
	if g.Current > 0 {
		sum.Round = g.Rounds[g.Current-1]
	}
	return sum
}

// NextRound advances the game to the next round. That includes incrementing the counter and setting
// a new round timer.
func (g *Game) NextRound() {
	g.Current++
	g.StreamerVote = 0
	g.voteHistory = make(map[string]bool)
	g.ChatVoteCount = [4]int{}
	g.RoundTimer = time.AfterFunc(g.RoundDuration, g.endRound)
}

func (g *Game) endRound() {
	if g == nil || g.connection == nil {
		return
	}

	if g.RoundTimer != nil && g.RoundTimer.Stop() {
		// If the round timer was running, the call to Stop will run this function again. To prevent
		// duplicates we exit here.
		return
	}
	g.RoundTimer = nil

	if g.connection.WS == nil {
		log.Println("tried to end round, but no websocket is not connected")
		return
	}

	roundSummary := struct {
		Type string `json:"type"`
		RoundSummary
	}{
		Type:         "ROUND_END",
		RoundSummary: g.GetRoundSummary(),
	}
	g.connection.WS.WriteJSON(roundSummary)
}

// GetRounds tries to get n questions from c. If c contains less than n questions, GetRounds returns
// all questions of c.
//
// The returned questions are in a randomized order.
func (c Category) GetRounds(n int) []*Round {
	if n == 0 {
		return []*Round{}
	}

	rand.Shuffle(len(c.Pool), func(i, j int) {
		c.Pool[i], c.Pool[j] = c.Pool[j], c.Pool[i]
	})

	var questions []*Question
	if n >= len(c.Pool) {
		questions = c.Pool
	} else {
		questions = c.Pool[:n]
	}

	var rounds = make([]*Round, 0, len(questions))
	for _, q := range questions {
		if q == nil {
			continue
		}
		round := q.ToRound()
		round.Category = c.Title
		rounds = append(rounds, &round)
	}
	return rounds
}

func (q Question) ToRound() Round {
	var answers []string

	// select one correct answer
	if len(q.Correct) > 1 {
		rand.Shuffle(len(q.Correct), func(i, j int) {
			q.Correct[i], q.Correct[j] = q.Correct[j], q.Correct[i]
		})
		answers = append(answers, q.Correct[rand.Intn(len(q.Correct)-1)])
	} else {
		answers = append(answers, q.Correct[0])
	}

	// select up to 3 wrong answers
	if len(q.Wrong) > 3 {
		rand.Shuffle(len(q.Wrong), func(i, j int) {
			q.Wrong[i], q.Wrong[j] = q.Wrong[j], q.Wrong[i]
		})
		answers = append(answers, q.Wrong[:3]...)
	} else {
		answers = append(answers, q.Wrong...)
	}

	var correct int
	rand.Shuffle(len(answers), func(i, j int) {
		answers[i], answers[j] = answers[j], answers[i]
		if i == correct || j == correct {
			correct = i + j - correct
		}
	})

	return Round{
		Question: q.Question,
		Answers:  answers,
		Correct:  correct + 1,
	}
}
