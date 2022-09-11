package gemini

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
)

type geminiPeer struct {
	server *geminiServer
	sock   net.Conn
	url    string
}

type geminiServer struct {
	listenSock net.Listener
}

/* ======================================[[ geminiPeer ]]======================================= */

func (server *geminiServer) newPeer(sock net.Conn) *geminiPeer {
	// status '20' is a SUCCESS status
	return &geminiPeer{server: server, sock: sock}
}

func (peer *geminiPeer) Kill() {
	peer.sock.Close()
}

func (peer *geminiPeer) Read(p []byte) (int, error) {
	return peer.sock.Read(p)
}

func (peer *geminiPeer) Write(p []byte) (int, error) {
	return peer.sock.Write(p)
}

func (peer *geminiPeer) readRequest() error {
	buf := make([]byte, 1026)
	length := 0

	// requests absolute url cannot be longer than 1024 bytes + <CR><LF> (2 bytes)
	for length < 1026 {
		sz, err := peer.Read(buf)
		if err != nil {
			return err
		}

		tmp := string(buf)
		peer.url += tmp
		length += sz

		// requests end with a <CR><LF>
		if length > 2 && buf[length-2] == '\r' && buf[length-1] == '\n' {
			break
		}
	}

	return nil
}

func (peer *geminiPeer) sendHeader(status int, meta string) {
	peer.Write([]byte(fmt.Sprintf("%d ", status)))
	peer.Write([]byte(meta))
	peer.Write([]byte{'\r', '\n'})
}

/* =====================================[[ geminiServer ]]====================================== */

func NewServer(port string, certFile string, keyFile string) (*geminiServer, error) {
	// load key pair && create config
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}

	// create listener socket
	log.Printf("listening on port %s\n", port)
	l, err := tls.Listen("tcp", ":"+port, config)
	if err != nil {
		return nil, err
	}

	return &geminiServer{listenSock: l}, nil
}

func (server *geminiServer) handlePeer(peer *geminiPeer) {
	log.Print("New peer!")
	defer peer.Kill()

	if err := peer.readRequest(); err != nil {
		log.Print(err)
	}

	log.Printf("got request URL: %s", peer.url)
	peer.sendHeader(40, "Stay tuned!")
}

func (server *geminiServer) Run() error {
	for {
		// block and wait until tls socket connects
		conn, err := server.listenSock.Accept()
		if err != nil {
			return err
		}

		// create peer and handle connection
		peer := server.newPeer(conn)
		go server.handlePeer(peer)
	}
}
