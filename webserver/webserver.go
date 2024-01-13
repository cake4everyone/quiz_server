package webserver

import (
	"fmt"
	logger "log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"
)

var (
	log            = logger.New(logger.Writer(), "[WEB] ", logger.LstdFlags|logger.Lmsgprefix)
	alreadyStarted = false
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

	addr := viper.GetString("webserver.address")
	var err error

	go func() {
		err = http.ListenAndServe(addr, initHandler())
		if failed != nil {
			alreadyStarted = false
			go failed(fmt.Errorf("webserver crashed: %v", err))
		}
	}()
	go func() {
		time.Sleep(1500 * time.Millisecond)
		if err == nil {
			log.Printf("Webserver started unter \"%s\"", addr)
			if started != nil {
				go started()
			}
		}
	}()
}

func initHandler() http.Handler {
	r := mux.NewRouter()
	r.NotFoundHandler = http.HandlerFunc(handle404)

	r.HandleFunc("/questions/fetch", handleFetchQuestions).Methods("PUT")

	return r
}

func handle404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
}
