package webserver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
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
	if body.Password != viper.GetString("connection.password") {
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

	log.Printf("logged in as %s", user.Username)

	token := uuid.New().String()
	activeAuth[token] = user.ID
	c := quiz.New(user.ID)
	if c == nil {
		log.Printf("Failed to create new quiz connection, but randomly generated uuid already exists: '%s'. Congrats to %s for drawing that uuid!", token, r.RemoteAddr)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var loginResponse struct {
		Username   string `json:"username"`
		TwitchName string `json:"twitch"`
		Token      string `json:"token"`
	}
	loginResponse.Username = user.Username
	loginResponse.Token = token

	// setting twitch connection
	twitchBot := twitchgo.New("", user.TwitchToken)

	gotTwitch := make(chan struct{})
	twitchBot.OnGlobalUserState(func(t *twitchgo.Twitch, userTags twitchgo.MessageTags) {
		loginResponse.TwitchName = userTags.DisplayName
		close(gotTwitch)
		t.SendMessage(userTags.DisplayName, "Welcome to Quiz4Everyone!")
	})

	err = twitchBot.Connect()
	if err != nil {
		if opErr, ok := err.(*net.OpError); ok {
			err = opErr.Unwrap()
		}
		log.Printf("Failed to connect to twitch: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	<-gotTwitch
	c.Twitch = twitchBot
	twitchBot.OnChannelMessage(c.OnTwitchChannelMessage)
	twitchBot.JoinChannel(loginResponse.TwitchName)

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
		log.Printf("Failed to upgrade to websocket communication")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.WS = wsConn

	handleWebsocket(c)
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
			w.WriteHeader(http.StatusBadRequest)
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

	round := c.Game.Rounds[c.Game.Current-1]
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

	round := c.Game.GetRoundSummary()
	b, err := json.Marshal(round)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if c.Game.Current >= len(c.Game.Rounds) {
		// if this is the last round send also game summary
		bGame, err := json.Marshal(c.Game.Summary)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		c.Game = nil

		bGame[0] = ','
		b = append(b[:len(b)-1], bGame...)
		w.Write(b)
		return
	}

	c.Game.Current++
	w.Write(b)
}
