/* gemini.go
extremely basic gemini server implementing the gemini protocol as described by:
	gemini://gemini.circumlunar.space/docs/specification.gmi
*/

package gemini

import (
	"crypto/tls"
	"fmt"
	"io"
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
	server   *GeminiServer
	sock     net.Conn
	rawURL   string
	hostname string
	path     string
	param    string
	uri      string
	params   map[string]string
}

type GeminiServer struct {
	listenSock net.Listener
}

type GeminiRequest struct {
	sock           *tls.Conn
	responseHeader string
	responseBody   string
}

/* ===================================[[ Helper Functions ]]==================================== */

// (can panic !)
func ParseURL(rawUrl string) (uri, hostname, path, param string) {
	// clean url, parse out the uri
	if i := strings.Index(rawUrl, "://"); i != -1 {
		uri = rawUrl[:i+3]  // eg. "gemini://"
		path = rawUrl[i+3:] // eg. "localhost/path/index.gmi"
	} else {
		uri = "gemini://"
		path = rawUrl
	}

	// split path into hostname and path
	if i := strings.Index(path, "/"); i != -1 {
		hostname = path[:i]
		path = path[i:]
	} else {
		hostname = path
		path = "/"
	}

	// grab parameter (if exists)
	if i := strings.Index(rawUrl, "?"); i != -1 {
		// decode param
		tparam, err := url.QueryUnescape(rawUrl[i+1:])
		if err != nil {
			panic("failed to decode param!")
		}

		// decode path
		tpath, err := url.PathUnescape(rawUrl[:i])
		if err != nil {
			panic("failed to decode path!")
		}

		// set
		param = tparam
		path = tpath
	}

	return
}

/* ======================================[[ GeminiPeer ]]======================================= */

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

// returns number of bytes read into p (can panic!)
func (peer *GeminiPeer) Read(p []byte) int {
	sz, err := peer.sock.Read(p)

	if err != nil {
		panic(err)
	}

	return sz
}

// writes bytes to tls connection (can panic !)
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

func (peer *GeminiPeer) readRequest() {
	buf := make([]byte, 1026)
	length := 0

	// requests absolute url cannot be longer than 1024 bytes + <CR><LF> (2 bytes)
	for length < 1026 {
		sz := peer.Read(buf[length:])

		// socket hangup (missing <CR><LF>)
		if sz == 0 {
			panic("malformed gemini request!")
		}

		length += sz
		// requests end with a <CR><LF>
		if length > 2 && buf[length-2] == '\r' && buf[length-1] == '\n' {
			break
		}
	}

	// -2 to remove the <CR><LF>
	peer.rawURL = string(buf[:length-2])

	// parse url
	peer.uri, peer.hostname, peer.path, peer.param = ParseURL(peer.rawURL)
}

func (peer *GeminiPeer) sendHeader(status int, meta string) {
	// <STATUS><SPACE><META><CR><LF>
	peer.Write([]byte(fmt.Sprintf("%d %s\r\n", status, meta)))

	log.Printf("%s <- STATUS %d '%s'", peer.GetAddr(), status, meta)
}

func (peer *GeminiPeer) GetAddr() string {
	return peer.sock.RemoteAddr().String()
}

// returns (param, isParam). if isParam is false, the peer did not post any parameter data
func (peer *GeminiPeer) GetParam() (string, bool) {
	return peer.param, strings.Compare(peer.param, "") != 0
}

// meta is the text that is prompted for the user (can panic !)
func (peer *GeminiPeer) SendInput(meta string) {
	peer.sendHeader(StatusInput, meta)
}

// meta is the text that is reported to the user (can panic !)
func (peer *GeminiPeer) SendError(meta string) {
	peer.sendHeader(StatusTemporaryFailure, meta)
}

// sends a StatusSuccess response header and the body (can panic !)
func (peer *GeminiPeer) SendBody(body *GeminiBody) {
	peer.sendHeader(StatusSuccess, "text/gemini")
	peer.Write([]byte(body.buf))
}

/* =====================================[[ GeminiRequest ]]===================================== */

// make a gemini request
func NewRequest(uri, hostname, port, path, param string) (req *GeminiRequest, err error) {
	config := tls.Config{
		ServerName:         hostname,
		InsecureSkipVerify: true,
	}

	// open tcp connection to gemini server
	conn, err := net.Dial("tcp", hostname+":"+port)
	if err != nil {
		return nil, err
	}

	// start tls handshake
	tlsConn := tls.Client(conn, &config)
	req = &GeminiRequest{sock: tlsConn}

	// error catching (for errors thrown from .Write() or .ReadHeaders())
	defer func() {
		// if someone threw a panic make sure we let the caller know
		if r, ok := recover().(error); ok {
			err = r
			req = nil
		}
	}()

	// write request
	req.Write([]byte(fmt.Sprintf("%s%s%s", uri, hostname, path)))

	// write parameter (if exists)
	if len(param) > 0 {
		req.Write([]byte(fmt.Sprintf("?%s", param)))
	}

	// write request terminator
	req.Write([]byte("\r\n"))

	// TODO: check if request is > 1026

	// read response headers
	req.readHeaders()

	// read body (TODO: if status is StatusSuccess "20")
	req.readBody()

	// success!
	return req, nil
}

func LazyRequest(url string) (result string, err error) {
	uri, hostname, path, param := ParseURL(url)
	req, err := NewRequest(uri, hostname, "1965", path, param)
	if err != nil {
		return "", err
	}

	return req.responseBody, nil
}

// simple wrapper to write raw data over the tls connection (can panic !)
func (req *GeminiRequest) Write(p []byte) {
	written := 0

	for written < len(p) {
		sz, err := req.sock.Write(p[written:])
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

// simple wrapper to read raw data over the tls connection
func (req *GeminiRequest) Read(p []byte) int {
	sz, err := req.sock.Read(p)

	// ignore EOF
	if err != nil && err != io.EOF {
		panic(fmt.Errorf("Read: %s", err))
	}

	return sz
}

// reads gemini response header (can panic !)
func (req *GeminiRequest) readHeaders() {
	buf := make([]byte, 1029)
	var length int = 0

	// response headers cannot be longer than status (2 bytes) + space (1 byte) + meta (1024 bytes max) + <CR><LF> (2 bytes)
	for length < 1029 {
		sz := req.Read(buf[length:])
		// socket hangup (missing <CR><LF>)
		if sz == 0 {
			panic("malformed gemini response!")
		}

		length += sz
		// response headers end with a <CR><LF>
		if length > 2 && buf[length-2] == '\r' && buf[length-1] == '\n' {
			break
		}
	}

	// save response header
	req.responseHeader = string(buf[:length-2])
}

// reads gemini response body (can panic!)
func (req *GeminiRequest) readBody() {
	buf := make([]byte, 1028)
	sz := 1

	// socket hangup marks the end of the response body (and exit condition)
	for sz != 0 {
		sz = req.Read(buf)

		// append read data into body
		req.responseBody += string(buf[0:])
	}
}

/* =====================================[[ GeminiServer ]]====================================== */

func NewServer(port, certFile, keyFile string) (*GeminiServer, error) {
	// load key pair && create config
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// create listener socket
	log.Printf("listening on port %s\n", port)
	l, err := tls.Listen("tcp", ":"+port, &config)
	if err != nil {
		return nil, err
	}

	return &GeminiServer{listenSock: l}, nil
}

// wrapper that reads the peer's request and dispatches the user-defined
// request handler. also has some simple error recovery for cleaning up the
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
