package webserver

import (
	"quiz_backend/quiz"
)

func handleWebsocket(c *quiz.Connection) {
	go wsRead(c)
}

func wsRead(c *quiz.Connection) {
	defer c.WS.Close()
	var err error
	for {
		var mt int
		var buf []byte
		mt, buf, err = c.WS.ReadMessage()
		if err != nil {
			break
		}

		log.Printf("=> '%x': %s", mt, string(buf))
		c.WS.WriteMessage(mt, append([]byte("Me can that too: "), buf...))
	}

	log.Printf("Error websocket read: %v", err)
}
