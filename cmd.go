package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"

	"github.com/alexcesaro/log"
	"github.com/alexcesaro/log/golog"
	"github.com/jessevdk/go-flags"
)

type Options struct {
	Verbose   []bool   `short:"v" long:"verbose" description:"Show verbose logging."`
	Identity  string   `short:"i" long:"identity" description:"Private key to identify server with." default:"~/.ssh/id_rsa"`
	Bind      string   `long:"bind" description:"Host and port to listen on." default:"0.0.0.0:22"`
	Admin     []string `long:"admin" description:"Fingerprint of pubkey to mark as admin."`
	Whitelist string   `long:"whitelist" description:"Optional file of pubkey fingerprints that are allowed to connect"`
}

var logLevels = []log.Level{
	log.Warning,
	log.Info,
	log.Debug,
}

func main() {
	options := Options{}
	parser := flags.NewParser(&options, flags.Default)

	p, err := parser.Parse()
	if err != nil {
		if p == nil {
			fmt.Print(err)
		}
		return
	}

	// Initialize seed for random colors
	RandomColorInit()

	// Figure out the log level
	numVerbose := len(options.Verbose)
	if numVerbose > len(logLevels) {
		numVerbose = len(logLevels)
	}

	logLevel := logLevels[numVerbose]
	logger = golog.New(os.Stderr, logLevel)

	privateKey, err := ioutil.ReadFile(options.Identity)
	if err != nil {
		logger.Errorf("Failed to load identity: %v", err)
		return
	}

	server, err := NewServer(privateKey)
	if err != nil {
		logger.Errorf("Failed to create server: %v", err)
		return
	}

	for _, fingerprint := range options.Admin {
		server.Op(fingerprint)
	}

	if options.Whitelist != "" {
		file, err := os.Open(options.Whitelist)
		if err != nil {
			logger.Errorf("Could not open whitelist file")
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			server.Whitelist(scanner.Text())
		}
	}

	// Construct interrupt handler
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	err = server.Start(options.Bind)
	if err != nil {
		logger.Errorf("Failed to start server: %v", err)
		return
	}

	<-sig // Wait for ^C signal
	logger.Warningf("Interrupt signal detected, shutting down.")
	server.Stop()
}
