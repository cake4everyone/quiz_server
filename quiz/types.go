package quiz

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

// Questions is the main list of all Categories and Questions
var Questions []Category
