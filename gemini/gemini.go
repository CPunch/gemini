/* gemini.go
extremely basic gemini server implementing the gemini protocol as described by:
	gemini://gemini.circumlunar.space/docs/specification.gmi
*/

package gemini

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
)

const (
	StatusInput              = 10
	StatusSuccess            = 20
	StatusRedirect           = 30
	StatusRedirectTemp       = 30
	StatusRedirectPerm       = 31
	StatusTemporaryFailure   = 40
	StatusUnavailable        = 41
	StatusPermanentFailure   = 50
	StatusNotFound           = 51
	StatusBadRequest         = 59
	StatusClientCertRequired = 60
)

type geminiPeer struct {
	server *geminiServer
	sock   net.Conn
	path   string
	param  string
	uri    string
	params map[string]string
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
	// catch any panics
	if r := recover(); r != nil {
		log.Printf("peer[%p] %s", peer, r)
	}

	peer.sock.Close()
	log.Printf("peer[%p] killed", peer)
}

func (peer *geminiPeer) Read(p []byte) (int, error) {
	return peer.sock.Read(p)
}

func (peer *geminiPeer) Write(p []byte) {
	written := 0

	for written < len(p) {
		sz, err := peer.sock.Write(p[written:])
		if err != nil {
			panic(err)
		}

		// if sz is 0, it means the socket has closed
		if sz == 0 {
			panic("premature socket hangup!")
		}

		written += sz
	}
}

func (peer *geminiPeer) readRequest() {
	buf := make([]byte, 1026)
	length := 0

	// requests absolute url cannot be longer than 1024 bytes + <CR><LF> (2 bytes)
	for length < 1026 {
		sz, err := peer.Read(buf[length:])
		if err != nil {
			panic(err)
		}

		length += sz
		// requests end with a <CR><LF>
		if length > 2 && buf[length-2] == '\r' && buf[length-1] == '\n' {
			break
		}
	}

	// -2 to remove the <CR><LF>
	rawURL := string(buf[:length-2])

	// clean url, parse out the uri
	if i := strings.Index(rawURL, "://"); i != -1 {
		peer.uri = rawURL[:i+3]  // eg. "gemini://"
		peer.path = rawURL[i+3:] // eg. "localhost/path/index.gmi"
	} else {
		peer.uri = "gemini://"
		peer.path = rawURL
	}

	// grab parameter (if exists)
	if i := strings.Index(peer.path, "?"); i != -1 {
		peer.param = url.QueryEscape(peer.path[i+1:])
		peer.path = peer.path[:i]
	}
}

func (peer *geminiPeer) sendHeader(status int, meta string) {
	// <STATUS><SPACE><META><CR><LF>
	peer.Write([]byte(fmt.Sprintf("%d %s\r\n", status, meta)))
}

/* =====================================[[ geminiServer ]]====================================== */

func NewServer(port, certFile, keyFile string) (*geminiServer, error) {
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

	peer.readRequest()
	log.Printf("got request URL: %s%s", peer.uri, peer.path)
	peer.sendHeader(StatusTemporaryFailure, "Stay tuned!")
}

func (server *geminiServer) Run() {
	for {
		// block and wait until tls socket connects
		conn, err := server.listenSock.Accept()
		if err != nil {
			log.Print("Listener socket: ", err)
			continue
		}

		// create peer and handle connection
		peer := server.newPeer(conn)
		go server.handlePeer(peer)
	}
}
