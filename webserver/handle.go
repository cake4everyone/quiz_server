package webserver

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"quiz_backend/quiz"

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
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read login body: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var login struct {
		Pass        string `json:"password"`
		TwitchToken string `json:"twitch"`
	}
	err = json.Unmarshal(body, &login)
	if err != nil {
		log.Printf("Failed to parse login body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	pass := viper.GetString("webserver.password")
	if login.Pass == "" || login.Pass != pass {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	if login.TwitchToken == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	token := uuid.New().String()
	c := quiz.New(token)
	if c == nil {
		log.Printf("Failed to create new quiz connection, but randomly generated uuid already exists: '%s'. Congrats to %s for drawing that uuid!", token, r.RemoteAddr)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var loginResponse struct {
		TwitchName string `json:"display_name"`
		Token      string `json:"token"`
	}
	loginResponse.Token = token

	// setting twitch connection
	twitchBot := twitchgo.New("", login.TwitchToken)

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

	body, err = json.Marshal(loginResponse)
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
