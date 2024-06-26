package webserver

import (
	"fmt"
	logger "log"
	"net/http"
	"quiz_backend/quiz"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

var (
	log            = logger.New(logger.Writer(), "[WEB] ", logger.LstdFlags|logger.Lmsgprefix)
	alreadyStarted = false
	upgrader       = websocket.Upgrader{}
)

// Start starts the webserver and begins to handle requests. Start takes two callback functions,
// started(), and failed(error). These function are called to send feedback to the parent function.
// failed is called when the webserver fails to start or when it chrashes at any point. started is
// called when it successfully started without a chrash.
func Start(started func(), failed func(error)) {
	if alreadyStarted {
		if failed != nil {
			failed(fmt.Errorf("webserver already started"))
		}
		return
	}
	alreadyStarted = true

	port := viper.GetInt("webserver.port")
	if port < 1 || port >= 1<<16 {
		failed(fmt.Errorf("webserver start: invalid port: %d", port))
		return
	}
	var err error

	go func() {
		err = http.ListenAndServe(":"+fmt.Sprint(port), initHandler())
		if failed != nil {
			alreadyStarted = false
			go failed(fmt.Errorf("webserver crashed: %v", err))
		}
	}()
	go func() {
		time.Sleep(1500 * time.Millisecond)
		if err == nil {
			log.Printf("Webserver started on port %d", port)
			if started != nil {
				go started()
			}
		}
	}()
}

func initHandler() http.Handler {
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(handle404)

	r.HandleFunc("/questions/fetch", handleFetchQuestions).Methods(http.MethodPut)
	r.HandleFunc("/login", login).Methods(http.MethodPost)

	r.HandleFunc("/logout", logout).Methods(http.MethodPost)
	r.HandleFunc("/chat", handleChat).Methods(http.MethodGet)

	r.HandleFunc("/category", handleCategory).Methods(http.MethodGet)

	r.HandleFunc("/game", handleGame)
	r.HandleFunc("/vote/streamer", handleStreamerVote).Methods(http.MethodPost)
	r.HandleFunc("/round", getRound).Methods(http.MethodGet)
	r.HandleFunc("/round/next", nextRound).Methods(http.MethodPost)

	return r
}

func handle404(w http.ResponseWriter, r *http.Request) {
	log.Printf("404: [%s] %s %s", r.Header.Get("X-Forwarded-For"), r.Method, r.RequestURI)
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(fmt.Sprintf("cant find %s", r.RequestURI)))
}

// activeAuth is a map from temporary tokens to a user id.
var activeAuth = make(map[string]string)

func isAuthorized(r *http.Request) (c *quiz.Connection, ok bool) {
	token := r.Header.Get("Authorization")
	token, found := strings.CutPrefix(token, "Q4E ")
	if !found {
		return nil, false
	}
	userID, found := activeAuth[token]
	if !found {
		return nil, false
	}
	return quiz.GetConnection(userID)
}
