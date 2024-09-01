package webserver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"quiz_backend/database"
	"quiz_backend/quiz"
	"strings"

	"github.com/google/uuid"
	"github.com/kesuaheli/twitchgo"
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
	if body.Password != viper.GetString("webserver.password") {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	log.Printf("Request to fetch questions by '%s'...", r.RemoteAddr)
	err = quiz.FetchQuestions()
	if err != nil {
		log.Printf("Error on request to fetch questions: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	auth, ok := strings.CutPrefix(auth, "Basic ")
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	b, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		log.Printf("Failed to decode auth: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	n := bytes.IndexByte(b, ':')
	if n == -1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	user := database.GetUserByCredentials(string(b[:n]), string(b[n+1:]))
	if user == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var loginResponse struct {
		Username   string `json:"username"`
		TwitchName string `json:"twitch"`
		Token      string `json:"token"`
	}
	loginResponse.Username = user.Username

	c := quiz.New(user.ID)
	if c == nil {
		// respond with token in activeAuths that matches user id
		var token, id string
		for token, id = range activeAuth {
			if id == user.ID {
				break
			}
		}
		if token == "" {
			log.Printf("ERROR: returned connection is nil")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Printf("relogged in as %s", user.Username)
		loginResponse.Token = token
		body, err := json.Marshal(loginResponse)
		if err != nil {
			log.Printf("Failed to marshal login response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			c.Close()
			return
		}
		w.Write(body)
		return
	}

	log.Printf("logged in as %s", user.Username)
	token := uuid.New().String()
	activeAuth[token] = user.ID

	loginResponse.Token = token

	c.Twitch = twitchgo.NewAPIOnly(
		viper.GetString("twitch.client_id"),
		viper.GetString("twitch.client_secret"),
	).SetAuthRefreshToken(
		user.TwitchToken,
	)
	tUser, err := c.Twitch.GetUser()
	if err != nil {
		log.Printf("GetUser error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.JoinTwitchChannel(tUser.Login)

	body, err := json.Marshal(loginResponse)
	if err != nil {
		log.Printf("Failed to marshal login response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		c.Close()
		return
	}
	w.Write(body)
}

func logout(w http.ResponseWriter, r *http.Request) {
	c, ok := isAuthorized(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	c.Close()
	w.WriteHeader(http.StatusOK)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	c, ok := isAuthorized(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to websocket communication: %v", err)
		return
	}
	c.WS = wsConn

	handleWebsocket(c)
}

func handleCategory(w http.ResponseWriter, r *http.Request) {
	type ResponseCategory struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Count       int    `json:"count"`
	}

	categories := make(map[string][]ResponseCategory)
	for _, group := range quiz.Categories {
		for _, cat := range group.Categories {
			responseCategory := ResponseCategory{
				Title:       cat.Title,
				Description: cat.Description,
				Count:       len(cat.Pool)}
			categories[group.Title] = append(categories[group.Title], responseCategory)
		}
	}

	b, err := json.Marshal(categories)
	if err != nil {
		log.Printf("Failed to marshal categories: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

func handleGame(w http.ResponseWriter, r *http.Request) {
	c, ok := isAuthorized(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = c.NewGame(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	case http.MethodGet:
		if c.Game == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b, err := json.Marshal(c.Game.Summary)
		if err != nil {
			log.Printf("Failed to marshal game summary: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(b)
		return
	case http.MethodDelete:
		c.Game = nil
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func handleStreamerVote(w http.ResponseWriter, r *http.Request) {
	c, ok := isAuthorized(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if c.Game == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if c.Game.StreamerVote != 0 {
		http.Error(w, "Streamer did already leave a vote!", http.StatusPreconditionFailed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var streamerVoteData struct {
		Vote string `json:"vote"`
	}
	err = json.Unmarshal(body, &streamerVoteData)
	if err != nil || streamerVoteData.Vote == "" {
		http.Error(w, "Not a valid json body. Need key 'vote'", http.StatusBadRequest)
		return
	}

	c.Game.StreamerVote = quiz.MsgToVote(streamerVoteData.Vote, c.Game)
	if c.Game.StreamerVote == 0 {
		http.Error(w, streamerVoteData.Vote+" is not a valid vote option", http.StatusBadRequest)
		return
	}

}

func getRound(w http.ResponseWriter, r *http.Request) {
	c, ok := isAuthorized(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if c.Game == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if c.Game.Current == 0 {
		http.Error(w, "no active round", http.StatusNotFound)
		return
	}

	round := *c.Game.Rounds[c.Game.Current-1]
	round.Correct = 0 // censoring correct answer
	b, err := json.Marshal(round)
	if err != nil {
		log.Printf("Failed to marshal current round: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(b)
}

func nextRound(w http.ResponseWriter, r *http.Request) {
	c, ok := isAuthorized(r)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if c.Game == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if c.Game.Current >= len(c.Game.Rounds) {
		// if this is the last round send game summary
		b, err := json.Marshal(c.Game.Summary)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		c.Game = nil
		w.Write(b)
		return
	}

	c.Game.NextRound()

	round := *c.Game.Rounds[c.Game.Current-1]
	round.Correct = 0 // censoring correct answer
	b, err := json.Marshal(round)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(b)
}
