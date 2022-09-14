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

type GeminiPeer struct {
	server *GeminiServer
	sock   net.Conn
	rawURL string
	path   string
	param  string
	uri    string
	params map[string]string
}

type GeminiServer struct {
	listenSock net.Listener
}

/* ======================================[[ geminiPeer ]]======================================= */

func (server *GeminiServer) newPeer(sock net.Conn) *GeminiPeer {
	return &GeminiPeer{server: server, sock: sock}
}

func (peer *GeminiPeer) Kill() {
	// catch any panics
	if r := recover(); r != nil {
		log.Printf("%s [ERR]: %s", peer.GetAddr(), r)
	}

	peer.sock.Close()
}

func (peer *GeminiPeer) Read(p []byte) (int, error) {
	return peer.sock.Read(p)
}

func (peer *GeminiPeer) Write(p []byte) {
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

func (peer *GeminiPeer) GetAddr() string {
	return peer.sock.LocalAddr().String()
}

func (peer *GeminiPeer) readRequest() {
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
	peer.rawURL = string(buf[:length-2])

	// clean url, parse out the uri
	if i := strings.Index(peer.rawURL, "://"); i != -1 {
		peer.uri = peer.rawURL[:i+3]  // eg. "gemini://"
		peer.path = peer.rawURL[i+3:] // eg. "localhost/path/index.gmi"
	} else {
		peer.uri = "gemini://"
		peer.path = peer.rawURL
	}

	// grab parameter (if exists)
	if i := strings.Index(peer.path, "?"); i != -1 {
		// decode param
		param, err := url.QueryUnescape(peer.path[i+1:])
		if err != nil {
			panic("failed to decode param!")
		}

		// decode path
		path, err := url.PathUnescape(peer.path[:i])
		if err != nil {
			panic("failed to decode path!")
		}

		// set
		peer.param = param
		peer.path = path
	}
}

func (peer *GeminiPeer) sendHeader(status int, meta string) {
	// <STATUS><SPACE><META><CR><LF>
	peer.Write([]byte(fmt.Sprintf("%d %s\r\n", status, meta)))

	log.Printf("%s <- STATUS %d '%s'", peer.GetAddr(), status, meta)
}

func (peer *GeminiPeer) SendError(meta string) {
	peer.sendHeader(StatusTemporaryFailure, meta)
}

func (peer *GeminiPeer) SendBody(body *GeminiBody) {
	peer.sendHeader(StatusSuccess, "text/gemini")
	peer.Write([]byte(body.buf))
}

/* =====================================[[ geminiServer ]]====================================== */

func NewServer(port, certFile, keyFile string) (*GeminiServer, error) {
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

	return &GeminiServer{listenSock: l}, nil
}

// wrapper that reads the peer's request and dispatches the user-defined
// request handler. also has some simple error recover for cleaning up the
// socket. request handlers are encouraged to use panic() if there is a
// non-peer related error. for request-related errors, use peer.SendError()
func (server *GeminiServer) handlePeer(peer *GeminiPeer, handler func(peer *GeminiPeer)) {
	defer peer.Kill()
	peer.readRequest()

	// log our transaction
	log.Printf("%s -> %s", peer.GetAddr(), peer.rawURL)

	// call our user-defined peer handler
	handler(peer)
}

func (server *GeminiServer) Run(peerRequest func(peer *GeminiPeer)) {
	for {
		// block and wait until tls socket connects
		conn, err := server.listenSock.Accept()
		if err != nil {
			log.Print("Listener socket: ", err)
			continue
		}

		// create peer and handle connection
		peer := server.newPeer(conn)
		go server.handlePeer(peer, peerRequest)
	}
}
