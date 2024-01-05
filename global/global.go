package global

import (
	"encoding/json"
	logger "log"
	"os"
	"quiz_backend/google"
	"quiz_backend/quiz"

	"github.com/spf13/viper"
)

var log = logger.New(logger.Writer(), "[QUIZ] ", logger.LstdFlags|logger.Lmsgprefix)

func FetchQuestions() error {
	log.Println("Getting Quiz from Google Spreadsheet...")

	var err error
	quiz.Categories, err = google.GetQuizFromSpreadsheet(viper.GetString("google.spreadsheetID"))
	if err != nil {
		return err
	}

	var questionCount, answerCount int
	for _, cat := range quiz.Categories {
		questionCount += len(cat.Pool)
		for _, q := range cat.Pool {
			answerCount += len(q.Correct)
			answerCount += len(q.Wrong)
		}
		data, err := json.MarshalIndent(cat, "", "	")
		if err != nil {
			log.Printf("Error marshaling category '%s': %v", cat.Title, err)
			continue
		}
		err = os.WriteFile("sheets/"+cat.Title+".json", data, 0644)
		if err != nil {
			log.Printf("Error writing json file of category '%s': %v", cat.Title, err)
		}
	}

	log.Printf("Got %d quiz categories with a total of %d questions and %d answers", len(quiz.Categories), questionCount, answerCount)

	return nil
}
