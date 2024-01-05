package quiz

import (
	logger "log"
)

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
