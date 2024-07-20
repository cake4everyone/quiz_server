package webserver

import (
	"quiz_backend/quiz"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

func handleWebsocket(c *quiz.Connection) {
	go wsRead(c)
	go keepAlive(c)
}

func wsRead(c *quiz.Connection) {
	defer c.Close()
	for {
		if c.WS == nil {
			return
		}
		mt, buf, err := c.WS.ReadMessage()
		if err != nil {
			log.Printf("Error websocket read: %v", err)
			break
		}

		log.Printf("=> '%x': %s", mt, string(buf))
		c.WS.WriteMessage(mt, append([]byte("Me can that too: "), buf...))
	}

}

func keepAlive(c *quiz.Connection) {
	c.SetLastResponse()
	c.WS.SetPongHandler(func(appData string) error {
		c.SetLastResponse()
		return nil
	})

	for {
		time.Sleep(viper.GetDuration("webserver.read_timeout") / 3)
		if c.WS == nil {
			return
		}
		if c.GetLastResponse() > viper.GetDuration("webserver.read_timeout") {
			log.Printf("App did not respond in time! Last response was %s ago. Closing connection...", c.GetLastResponse())
			c.Close()
			return
		}
		err := c.WS.WriteControl(websocket.PingMessage, []byte("PING"), time.Now().Add(viper.GetDuration("webserver.read_timeout")))
		if err != nil {
			log.Printf("Failed to send websocket ping message: %v | Closing connection...", err)
			c.Close()
		}
	}
}
