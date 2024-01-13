package quiz

import (
	"encoding/json"
	"os"
	"time"

	"github.com/spf13/viper"
)

var (
	lastFetch time.Time
)

func FetchQuestions() (err error) {
	const timeout = 30 * time.Second
	if time.Now().Add(timeout).Before(lastFetch) {
		return nil
	}
	lastFetch = time.Now()

	log.Println("Getting Quiz from Google Spreadsheet...")
	Categories, err = ParseFromGoogleSheets(viper.GetString("google.spreadsheetID"))
	if err != nil {
		return err
	}

	var questionCount, answerCount int
	for _, cat := range Categories {
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

	log.Printf("Got %d quiz categories with a total of %d questions and %d answers", len(Categories), questionCount, answerCount)

	return nil
}
