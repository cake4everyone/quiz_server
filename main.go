package main

import (
	"context"
	"fmt"
	logger "log"
	"os"
	"os/signal"
	"quiz_backend/config"
	"quiz_backend/global"
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

	fmt.Println("Press Ctrl+C to exit")
	<-ctx.Done()
	fmt.Println()
	fmt.Println("Shutting down")
}
