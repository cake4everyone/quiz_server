package google

import (
	"context"
	"fmt"
	logger "log"
	"quiz_backend/quiz"

	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var log = logger.New(logger.Writer(), "[GOOGLE] ", logger.LstdFlags|logger.Lmsgprefix)

func GetQuizFromSpreadsheet(ID string) ([]quiz.Category, error) {
	sService, err := sheets.NewService(context.Background(), option.WithAPIKey(viper.GetString("google.api_key")))
	if err != nil {
		return []quiz.Category{}, err
	}

	sCall := sService.Spreadsheets.Get(ID)
	sCall.IncludeGridData(true)
	sCall.Ranges()
	s, err := sCall.Do()
	if err != nil {
		return []quiz.Category{}, err
	}

	var categories []quiz.Category

	for _, s := range s.Sheets {
		if len(s.Data) == 0 {
			continue
		}

		category := parseSheet(s)
		if len(category.Pool) > 0 {
			categories = append(categories, category)
		}
	}
	return categories, nil
}

func parseSheet(s *sheets.Sheet) quiz.Category {
	category := quiz.Category{Title: s.Properties.Title}

	for rowNum, row := range s.Data[0].RowData {
		if rowNum <= 1 {
			if rowNum == 0 && len(row.Values) > 0 {
				// get description from first cell in first row
				category.Description = row.Values[0].FormattedValue
			}
			// ignore rest of 1st + 2nd row
			continue
		}

		question, err := getQuestionFromRow(row)
		if err != nil {
			log.Printf("Warn: could not get question from %s row %d: %v", s.Properties.Title, rowNum, err)
			continue
		}
		if question != nil {
			category.Pool = append(category.Pool, question)
		}
	}

	return category
}

func getQuestionFromRow(row *sheets.RowData) (*quiz.Question, error) {

	qq := &quiz.Question{}

	for cellNum, cell := range row.Values {
		// skip empty cells
		if cell == nil || cell.FormattedValue == "" {
			continue
		}

		// only read contents of the first cell and save it as the question
		if cellNum == 0 {
			qq.Question = cell.FormattedValue
			continue
		}

		var (
			color   *sheets.Color
			cFormat *sheets.CellFormat
		)
		// check formatting and color
		if cell.EffectiveFormat != nil {
			cFormat = cell.EffectiveFormat
		} else {
			cFormat = cell.UserEnteredFormat
		}
		if cFormat == nil {
			log.Printf("Warn: no format set in 'question '%s' answer %d ('%s')", qq.Question, cellNum, cell.FormattedValue)
			continue
		}

		if cFormat.BackgroundColorStyle != nil {
			color = cFormat.BackgroundColorStyle.RgbColor
		} else {
			color = cFormat.BackgroundColor
		}
		if color == nil {
			log.Printf("Warn: no color set in 'question '%s' answer %d ('%s')", qq.Question, cellNum, cell.FormattedValue)
			continue
		}

		if color.Green > color.Red {
			qq.Correct = append(qq.Correct, cell.FormattedValue)
		} else {
			qq.Wrong = append(qq.Wrong, cell.FormattedValue)
		}
	}

	// validation
	if qq.Question == "" {
		if len(qq.Correct) == 0 && len(qq.Wrong) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("missing question")
	}
	if len(qq.Correct) == 0 {
		return nil, fmt.Errorf("need at least one correct answer")
	}
	if len(qq.Wrong) == 0 {
		return nil, fmt.Errorf("need at least one incorrect answer")
	}

	return qq, nil
}
