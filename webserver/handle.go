package webserver

import (
	"encoding/json"
	"io"
	"net/http"
	"quiz_backend/global"

	"github.com/spf13/viper"
)

func handleFetchQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	err = json.Unmarshal(data, &body)
	if err != nil {
		log.Printf("Error unmarshaling body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"message\":\"invalid body\"}"))
		return
	}

	if body.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"message\":\"missing required key \\\"password\\\"!\"}"))
		return
	}
	if body.Password != viper.GetString("connection.password") {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	log.Printf("Request to fetch questions by '%s'...", r.RemoteAddr)
	err = global.FetchQuestions()
	if err != nil {
		log.Printf("Error on request to fetch questions: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
