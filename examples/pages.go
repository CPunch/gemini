package main

import (
	"flag"
	"log"

	"github.com/CPunch/gemini"
)

func handleIndex(peer *gemini.GeminiPeer) {
	body := gemini.NewBody()
	body.AddLinkLine("/hi", "click me!")
	peer.SendBody(body)
}

func handleHi(peer *gemini.GeminiPeer) {
	body := gemini.NewBody()
	body.AddHeader("Stay Tuned!")
	peer.SendBody(body)
}

func main() {
	// get command line flags
	port := flag.String("port", "1965", "listening port")
	certFile := flag.String("cert", "cert.pem", "certificate PEM file")
	keyFile := flag.String("key", "key.pem", "key PEM file")
	flag.Parse()

	// create server
	server, err := gemini.NewServer(*port, *certFile, *keyFile)
	if err != nil {
		log.Fatal(err)
	}

	// create path handler
	pHndler := gemini.NewHandler()
	pHndler.AddHandler("/", handleIndex)
	pHndler.AddHandler("/hi", handleHi)
	server.Run(pHndler.HandlePeer)
}
