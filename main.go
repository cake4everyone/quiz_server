package main

import (
	"context"
	"fmt"
	logger "log"
	"os"
	"os/signal"
	"quiz_backend/config"
	"quiz_backend/connection"
	"quiz_backend/global"
	"quiz_backend/webserver"
	"syscall"
)

var log = logger.New(logger.Writer(), "[MAIN] ", logger.LstdFlags|logger.Lmsgprefix)

func init() {
	config.Load()
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT)
	defer cancel()

	err := global.FetchQuestions()
	if err != nil {
		log.Printf("Error getting quiz: %v", err)
		os.Exit(-1)
	}

	webserver.Start(nil, func(err error) {
		log.Printf("Error %v", err)
		cancel()
	})

	err = connection.Start()
	if err != nil {
		log.Printf("Error %v", err)
		cancel()
	}

	fmt.Println()
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()
	<-ctx.Done()
	fmt.Println()
	fmt.Println("Shutting down")
}
