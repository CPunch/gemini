package main

import (
	"flag"
	"log"

	"github.com/CPunch/gem/gemini"
)

func handleRequest(peer *gemini.GeminiPeer) {
	peer.SendError("Stay tuned!")
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

	server.Run(handleRequest)
}
