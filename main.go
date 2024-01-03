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

	log.Printf("Got %d quiz categories", len(categories))

	for _, cat := range categories {
		data, err := json.MarshalIndent(cat, "", "	")
		if err != nil {
			log.Printf("Error marshaling category '%s': %v", cat.Title, err)
			continue
		}
		os.WriteFile("sheets/"+cat.Title+".json", data, 0644)
	}

}
