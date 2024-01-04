package connection

import (
	"fmt"
	logger "log"
	"net"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Connection struct {
	tcp     *net.TCPListener
	conn    *net.TCPConn
	started time.Time

	pass string
}

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
		conn.Write([]byte("Verified, welcome!\n"))
		go handleTCP(conn)
	}
}

func handleTCP(conn *net.TCPConn) {
	conn.SetDeadline(time.Time{})
	for {
		msg, err := readTCP(conn)
		if err != nil {
			log.Printf("failed to read from '%s' tcp: %v", conn.RemoteAddr().String(), err)
			break
		}
		go handleTCPMsg(msg)
	}
	log.Printf("connection with '%s' closed!", conn.RemoteAddr().String())
	conn = nil
}

func readTCP(conn *net.TCPConn) (string, error) {
	var buf []byte = make([]byte, 1024)
	n, err := conn.Read(buf)
	return strings.TrimRight(string(buf[:n]), "\n"), err
}

func handleTCPMsg(msg string) {
	log.Printf("new message: %s", msg)
}
