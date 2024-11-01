package quiz

import (
	logger "log"
	"math"
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
	Summary       *GameSummary
}

type GameSummary struct {
	StreamerPoints int `json:"streamer_points"`
	StreamerWon    int `json:"streamer_won"`
	ChatPoints     int `json:"chat_points"`
	ChatWon        int `json:"chat_won"`
}

type CategoryGroupDefinition struct {
	ID         string               `json:"id"`
	Title      string               `json:"title"`
	IsDev      bool                 `json:"-"`
	IsRelease  bool                 `json:"-"`
	Categories []CategoryDefinition `json:"categories,omitempty"`
}

type CategoryGroup struct {
	CategoryGroupDefinition
	Categories []Category `json:"categories"`
}

func (cg CategoryGroup) GetDefinition() CategoryGroupDefinition {
	cg.CategoryGroupDefinition.Categories = make([]CategoryDefinition, len(cg.Categories))
	for i, c := range cg.Categories {
		cg.CategoryGroupDefinition.Categories[i] = c.GetDefinition()
	}
	return cg.CategoryGroupDefinition
}

type CategoryDefinition struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Count int    `json:"count,omitempty"`
}

type Category struct {
	CategoryDefinition
	Pool []*Question `json:"pool"`
}

func (c Category) GetDefinition() CategoryDefinition {
	c.CategoryDefinition.Count = len(c.Pool)
	return c.CategoryDefinition
}

type Question struct {
	Question DisplayableContent   `json:"question"`
	Correct  []DisplayableContent `json:"correct"`
	Wrong    []DisplayableContent `json:"wrong"`
}

type DisplayableContent struct {
	Type  ContentType `json:"type"`
	Text  string      `json:"text"`
	Media []byte      `json:"-"`
}

type ContentType uint8

const (
	CONTENTTEXT ContentType = iota
	CONTENTIMAGE
)

type Round struct {
	Question DisplayableContent      `json:"question"`
	Answers  []DisplayableContent    `json:"answers"`
	Correct  int                     `json:"correct,omitempty"`
	Current  int                     `json:"current_round"`
	Max      int                     `json:"max_round"`
	Group    CategoryGroupDefinition `json:"group"`
	Category CategoryDefinition      `json:"category"`
}

type RoundSummary struct {
	*Round
	StreamerPoints int    `json:"streamer_points"`
	StreamerVote   int    `json:"streamer_vote"`
	ChatPoints     int    `json:"chat_points"`
	ChatVote       int    `json:"chat_vote"`
	ChatVoteCount  [4]int `json:"chat_vote_count"`
}

type categoryGroups map[int]CategoryGroup
type categoryGroupDefinitions map[int]CategoryGroupDefinition

// Categories is the main list of all Categories and Groups
var Categories categoryGroups

var log = logger.New(logger.Writer(), "[WEB] ", logger.LstdFlags|logger.Lmsgprefix)

func (cg categoryGroups) GetCategoryByID(id string) Category {
	for _, group := range cg {
		for _, c := range group.Categories {
			if c.ID == id {
				return c
			}
		}
	}
	return Category{}
}

func (cg categoryGroups) GetGroupByID(id string) CategoryGroup {
	for _, group := range cg {
		if group.ID == id {
			return group
		}
	}
	return CategoryGroup{}
}

func (cg categoryGroups) GetDefinition() (definitions map[int]CategoryGroupDefinition) {
	definitions = make(map[int]CategoryGroupDefinition, len(cg))
	for color, group := range cg {
		definitions[color] = group.GetDefinition()
	}
	return definitions
}

func (cg categoryGroups) ShuffleCategories(groupID string, amount int, categories map[string]int) {
	if amount == 0 {
		return
	}
	group := cg.GetGroupByID(groupID)
	if group.ID == "" {
		return
	}

	if categories == nil {
		categories = make(map[string]int)
	}
	var shuffleSelection = make([]Category, 0, len(group.Categories))
	for _, category := range group.Categories {
		if categories[category.ID] < len(category.Pool) {
			shuffleSelection = append(shuffleSelection, category)
		}
	}
	for range amount {
		categoryIndex := rand.Intn(len(shuffleSelection))
		category := shuffleSelection[categoryIndex]

		categories[category.ID]++
		if categories[category.ID] > len(category.Pool) {
			shuffleSelection = append(shuffleSelection[:categoryIndex], shuffleSelection[categoryIndex+1:]...)
		}
	}
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

	// determine winner
	correct := g.Rounds[g.Current-1].Correct
	const roundPoints = 5

	if g.StreamerVote == correct {
		g.Summary.StreamerPoints += roundPoints
		g.Summary.StreamerWon++
	}
	chatCorrect := g.ChatVoteCount[correct-1]
	g.ChatVote = correct
	var totalVotes int
	for i, votes := range g.ChatVoteCount {
		totalVotes += votes
		if votes > chatCorrect {
			g.ChatVote = i + 1
		}
	}
	if totalVotes > 0 && g.ChatVote == correct {
		g.Summary.ChatPoints += roundPoints
		g.Summary.ChatWon++
	} else if totalVotes == 0 {
		g.ChatVote = 0
	}

	// send to ws
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
		round.Category = c.GetDefinition()
		rounds = append(rounds, &round)
	}
	return rounds
}

func (q Question) ToRound() Round {
	var answers []DisplayableContent

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
	}
	num_wrong := int(math.Min(float64(cap(q.Wrong)), 3))
	answers = append(answers, q.Wrong[:num_wrong]...)

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
