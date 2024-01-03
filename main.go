package main

import (
	"encoding/json"
	logger "log"
	"os"
	"quiz_backend/config"
	"quiz_backend/google"

	"github.com/spf13/viper"
)

var log = logger.New(logger.Writer(), "[MAIN] ", logger.LstdFlags|logger.Lmsgprefix)

func init() {
	config.Load()
}

func main() {
	log.Println("Getting Quiz from Google Spreadsheet...")
	categories, err := google.GetQuizFromSpreadsheet(viper.GetString("google.spreadsheetID"))
	if err != nil {
		log.Printf("Error getting quiz from Google Spreadsheet: %v", err)
		os.Exit(-1)
	}

	var questionCount, answerCount int
	for _, cat := range categories {
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

	log.Printf("Got %d quiz categories with a total of %d questions and %d answers", len(categories), questionCount, answerCount)

}
