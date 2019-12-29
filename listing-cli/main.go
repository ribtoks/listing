package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	modeFlag           = flag.String("mode", "", "Execution mode: subscribe|unsubscribe|export|import|delete")
	urlFlag            = flag.String("url", "", "Base URL to the listing API")
	emailFlag          = flag.String("email", "", "Email for subscribe|unsubscribe")
	authTokenFlag      = flag.String("auth-token", "", "Auth token for admin access")
	secretFlag         = flag.String("secret", "", "Secret for email salt")
	newsletterFlag     = flag.String("newsletter", "", "Newsletter for subscribe|unsubscribe")
	formatFlag         = flag.String("format", "table", "Ouput format of subscribers: csv|tsv|table|raw")
	nameFlag           = flag.String("name", "", "(optional) Name for subscribe")
	logPathFlag        = flag.String("l", "listing-cli.log", "Absolute path to log file")
	stdoutFlag         = flag.Bool("stdout", false, "Log to stdout and to logfile")
	helpFlag           = flag.Bool("help", false, "Print help")
	dryRunFlag         = flag.Bool("dry-run", false, "Simulate selected action")
	noUnconfirmedFlag  = flag.Bool("no-unconfirmed", false, "Do not export unconfirmed emails")
	noUnsubscribedFlag = flag.Bool("no-unsubscribed", false, "Do not export unsubscribed emails")
)

const (
	appName         = "listing-cli"
	modeSubscribe   = "subscribe"
	modeUnsubscribe = "unsubscribe"
	modeExport      = "export"
	modeImport      = "import"
	modeDelete      = "delete"
)

var (
	supportedModes = [...]string{modeSubscribe, modeUnsubscribe, modeExport, modeImport, modeDelete}
)

func main() {
	err := parseFlags()
	if err != nil {
		flag.PrintDefaults()
		log.Fatal(err.Error())
	}

	logfile, err := setupLogging()
	if err != nil {
		defer logfile.Close()
	}

	client := &listingClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		printer:        NewPrinter(),
		url:            *urlFlag,
		authToken:      *authTokenFlag,
		secret:         *secretFlag,
		dryRun:         *dryRunFlag,
		noUnconfirmed:  *noUnconfirmedFlag,
		noUnsubscribed: *noUnsubscribedFlag,
	}

	switch *modeFlag {
	case modeExport:
		{
			err = client.export(*newsletterFlag)
		}
	case modeSubscribe:
		{
			err = client.subscribe(*emailFlag, *newsletterFlag, *nameFlag)
		}
	default:
		fmt.Printf("Mode %v is not supported yet", *modeFlag)
	}
	if err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}

func setupLogging() (f *os.File, err error) {
	f, err = os.OpenFile(*logPathFlag, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", *logPathFlag)
		return nil, err
	}

	if *stdoutFlag || *dryRunFlag {
		mw := io.MultiWriter(os.Stdout, f)
		log.SetOutput(mw)
	} else {
		log.SetOutput(f)
	}

	log.Println("------------------------------")
	log.Println(appName + " log started")

	return f, err
}

func parseFlags() error {
	flag.Parse()

	if *modeFlag == "" {
		return errors.New("Mode is a required parameter")
	}

	found := false
	for _, m := range supportedModes {
		if m == *modeFlag {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("Mode %v is not supported", *modeFlag)
	}

	return nil
}

func NewPrinter() Printer {
	switch *formatFlag {
	case "table":
		return NewTablePrinter(*secretFlag)
	case "csv":
		return NewCSVPrinter(*secretFlag)
	case "tsv":
		return NewTSVPrinter(*secretFlag)
	case "raw":
		return NewRawPrinter()
	default:
		return NewTablePrinter(*secretFlag)
	}
}
