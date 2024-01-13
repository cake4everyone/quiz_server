package google

import (
	"context"
	logger "log"

	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var log = logger.New(logger.Writer(), "[GOOGLE] ", logger.LstdFlags|logger.Lmsgprefix)

func GetQuizFromSpreadsheet(ID string) ([]*sheets.Sheet, error) {
	sService, err := sheets.NewService(context.Background(), option.WithAPIKey(viper.GetString("google.api_key")))
	if err != nil {
		return nil, err
	}

	sCall := sService.Spreadsheets.Get(ID)
	sCall.IncludeGridData(true)
	sCall.Ranges()
	s, err := sCall.Do()
	if err != nil {
		return nil, err
	}
	return s.Sheets, nil
}
