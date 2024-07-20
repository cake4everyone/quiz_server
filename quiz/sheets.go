package quiz

import (
	"fmt"
	"math"
	"quiz_backend/google"

	"google.golang.org/api/sheets/v4"
)

func ParseFromGoogleSheets(ID string) (categories map[int]CategoryGroup, err error) {
	sheets, err := google.GetQuizFromSpreadsheet(ID)
	if err != nil {
		return nil, err
	}

	categories = make(map[int]CategoryGroup)
	for _, s := range sheets {
		if len(s.Data) == 0 {
			continue
		}
		if s.Properties.Title == "categories" {
			parseCategoryGroups(s, categories)
			continue
		}

		category := parseSheet(s)
		if len(category.Pool) == 0 {
			continue
		}
		hexColor := intFromColor(s.Properties.TabColorStyle.RgbColor)
		categoryGroup := categories[hexColor]
		categoryGroup.Categories = append(categoryGroup.Categories, category)
		categories[hexColor] = categoryGroup
	}
	return categories, nil
}

func parseCategoryGroups(s *sheets.Sheet, categories map[int]CategoryGroup) {
	for rowNum, row := range s.Data[0].RowData {
		if rowNum <= 1 {
			continue
		}
		if len(row.Values) < 4 {
			log.Printf("Warn: could not category group from row %d: too few values", rowNum)
			continue
		}
		if row.Values[1].FormattedValue == "" {
			continue
		}
		color, err := getColorFromCell(row.Values[0])
		if err != nil {
			log.Printf("Warn: could not category group from row %d: %v", rowNum, err)
			continue
		}

		hexColor := intFromColor(color)
		categoryGroup := categories[hexColor]
		categoryGroup.Title = row.Values[1].FormattedValue
		categoryGroup.IsDev = row.Values[2].FormattedValue == "TRUE"
		categoryGroup.IsRelease = row.Values[3].FormattedValue == "TRUE"
		categories[hexColor] = categoryGroup
	}
}

func parseSheet(s *sheets.Sheet) Category {
	category := Category{Title: s.Properties.Title}

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

func getQuestionFromRow(row *sheets.RowData) (qq *Question, err error) {
	qq = &Question{}
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

		color, err := getColorFromCell(cell)
		if err != nil {
			log.Printf("Warn: in 'question '%s' answer %d ('%s'): %v", qq.Question, cellNum, cell.FormattedValue, err)
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

func getColorFromCell(cell *sheets.CellData) (color *sheets.Color, err error) {
	var format *sheets.CellFormat
	if cell.EffectiveFormat != nil {
		format = cell.EffectiveFormat
	} else {
		format = cell.UserEnteredFormat
	}
	if format == nil {
		return nil, fmt.Errorf("no format set")
	}

	if format.BackgroundColorStyle != nil {
		color = format.BackgroundColorStyle.RgbColor
	} else {
		color = format.BackgroundColor
	}
	if color == nil {
		return nil, fmt.Errorf("no color set")
	}
	return color, nil
}

// hexFromColor returns the RGBA color in a hex format in lowercase letters (0xrrggbbaa)
func hexFromColor(color *sheets.Color) string {
	return fmt.Sprintf("0x%02x%02x%02x%02x",
		uint8(math.Ceil(color.Red*255)),
		uint8(math.Ceil(color.Green*255)),
		uint8(math.Ceil(color.Blue*255)),
		uint8(math.Ceil(color.Alpha*255)),
	)
}

// intFromColor returns the RGBA color as an integer.
func intFromColor(color *sheets.Color) int {
	return int(math.Ceil(color.Red*255))&0xFF<<24 |
		int(math.Ceil(color.Green*255))&0xFF<<16 |
		int(math.Ceil(color.Blue*255))&0xFF<<8 |
		int(math.Ceil(color.Alpha*255))&0xFF
}
