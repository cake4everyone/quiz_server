package connection

import (
	"encoding/json"
	"fmt"
	logger "log"
	"net"
	"quiz_backend/quiz"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Connection struct {
	tcp     *net.TCPListener
	conn    *net.TCPConn
	started time.Time
	Game    *quiz.Game
}

// AllConnections is a list of all currently active connections
var AllConnections []*Connection

var log = logger.New(logger.Writer(), "[TCP] ", logger.LstdFlags|logger.Lmsgprefix)

func Start() error {
	tcp, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: viper.GetInt("connection.port")})
	if err != nil {
		return fmt.Errorf("failedto create new TCP listener: %v", err)
	}

	_, port, err := net.SplitHostPort(tcp.Addr().String())
	if err != nil {
		tcp.Close()
		return fmt.Errorf("failed to get port from address: %v", err)
	}

	go listen(tcp)

	log.Printf("Started connection listen on port %s ", port)
	return nil
}

func listen(tcp *net.TCPListener) {
	for {
		conn, err := tcp.AcceptTCP()
		if err != nil {
			log.Printf("Failed to accept tcp connection: %v", err)
			break
		}
		log.Printf("<%s> new connection", conn.RemoteAddr())

		timeout := time.Now().Add(5 * time.Minute)
		conn.SetDeadline(timeout)

		var buf []byte = make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("<%s> read error: %v", conn.RemoteAddr().String(), err)
			conn.Close()
			continue
		}
		msg := strings.TrimRight(string(buf[:n]), "\n")
		n = len(msg)
		if msg != viper.GetString("connection.password") {
			conn.Close()
			log.Printf("<%s> verification failed!", conn.RemoteAddr().String())
			continue
		}
		log.Printf("<%s> verified connection!", conn.RemoteAddr().String())
		var c Connection = Connection{
			tcp:     tcp,
			conn:    conn,
			started: time.Now(),
		}
		AllConnections = append(AllConnections, &c)
		c.Reply(CommandINFO, []byte("Verified, welcome!"))
		go c.handleTCP()
	}
}

func (c *Connection) handleTCP() {
	c.conn.SetDeadline(time.Time{})
	var err error
	for {
		var msg string
		msg, err = c.readTCP()
		if err != nil {
			break
		}
		if msg != "" {
			go c.handleTCPMsg(msg)
		}
	}
	log.Printf("connection with '%s' closed: %v", c.conn.RemoteAddr().String(), err)
	for i, savedConn := range AllConnections {
		if savedConn == c {
			// overwrite the index of the one to remove with the last element
			AllConnections[i] = AllConnections[len(AllConnections)-1]
			AllConnections = AllConnections[:len(AllConnections)-1]
		}
	}
	c.conn.Close()
	c.conn = nil
}

func (c *Connection) readTCP() (string, error) {
	var buf []byte = make([]byte, 1024)
	n, err := c.conn.Read(buf)
	return strings.TrimRight(string(buf[:n]), "\n"), err
}

func (c *Connection) handleTCPMsg(msg string) {
	log.Printf("new message: %s", msg)
	n := strings.IndexByte(msg, ' ')
	var cmd, data string
	if n > 0 {
		cmd = msg[:n]
		data = strings.TrimLeft(msg[n:], " ")
	} else {
		cmd = msg
	}

	switch TCPCommand(cmd) {
	case CommandAUTH:
		c.handleAUTH(data)
	case CommandEND:
		c.handleEND(data)
	case CommandGAME:
		c.handleGAME(data)
	case CommandNEXT:
		c.handleNEXT(data)
	case CommandVOTE:
		c.handleVOTE(data)
	}
}

func (c Connection) Reply(cmd TCPCommand, data []byte) {
	if len(data) > 0 {
		data = append([]byte(cmd+" "), data...)
	} else {
		data = []byte(cmd)
	}
	data = append(data, '\n')
	c.conn.Write(data)
}

func (c Connection) ReplyERR(msg string) {
	c.Reply(CommandERROR, []byte("{\"error\":\""+msg+"\"}"))
}

func (c Connection) ReplyERRf(format string, a ...any) {
	c.ReplyERR(fmt.Sprintf(format, a...))
}

func (c *Connection) sendNextRound() {
	c.Game.Current++
	b, err := json.Marshal(c.Game.Rounds[c.Game.Current-1])
	if err != nil {
		log.Printf("Failed to marshal round data: %v", err)
		c.ReplyERR("Internal Server Error")
		return
	}
	c.Reply(CommandNEXT, b)
}
