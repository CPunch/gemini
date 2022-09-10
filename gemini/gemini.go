package gemini

import (
	"crypto/tls"
	"log"
	"net"
)

type geminiPeer struct {
	server *geminiServer
	sock   net.Conn
	url    string
	status int
	body   string
}

type geminiServer struct {
	listenSock net.Listener
}

/* ======================================[[ geminiPeer ]]======================================= */

func (server *geminiServer) newPeer(sock net.Conn) *geminiPeer {
	// status '20' is a SUCCESS status
	return &geminiPeer{server: server, sock: sock, status: 20}
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
	// TODO
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
