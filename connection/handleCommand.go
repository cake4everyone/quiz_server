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
	category := quiz.Categories.GetCategoryByName("Testkeks")

	question := category.Pool[0]
	if len(category.Pool) > 1 {
		rand.Shuffle(len(category.Pool), func(i, j int) {
			category.Pool[i], category.Pool[j] = category.Pool[j], category.Pool[i]
		})
		question = category.Pool[rand.Intn(len(category.Pool)-1)]
	}
	var answers []string

	// select one correct answer
	if len(question.Correct) > 1 {
		rand.Shuffle(len(question.Correct), func(i, j int) {
			question.Correct[i], question.Correct[j] = question.Correct[j], question.Correct[i]
		})
		answers = append(answers, question.Correct[rand.Intn(len(question.Correct)-1)])
	} else {
		answers = append(answers, question.Correct[0])
	}

	// select up to 3 wrong answers
	if len(question.Wrong) > 3 {
		rand.Shuffle(len(question.Wrong), func(i, j int) {
			question.Wrong[i], question.Wrong[j] = question.Wrong[j], question.Wrong[i]
		})
		answers = append(answers, question.Wrong[:3]...)
	} else {
		answers = append(answers, question.Wrong...)
	}

	rand.Shuffle(len(answers), func(i, j int) {
		answers[i], answers[j] = answers[j], answers[i]
	})

	round := roundData{
		Question: question.Question,
		Category: category.Title,
		Answers:  answers,
	}

	b, err := json.Marshal(round)
	if err != nil {
		log.Printf("failed to marshal round data: %v", err)
		return
	}

	b = append([]byte(CommandNEXT+" "), b...)

	c.conn.Write(b)
}

func (c *Connection) handleNEXT(data string) {

}

func (c *Connection) handleVOTE(data string) {

}
