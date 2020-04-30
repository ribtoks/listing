package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var (
	modeFlag             = flag.String("mode", "", "Execution mode: subscribe|unsubscribe|export|import|delete")
	urlFlag              = flag.String("url", "", "Base URL to the listing API")
	emailFlag            = flag.String("email", "", "Email for subscribe|unsubscribe")
	authTokenFlag        = flag.String("auth-token", "", "Auth token for admin access")
	secretFlag           = flag.String("secret", "", "Secret for email salt")
	newsletterFlag       = flag.String("newsletter", "", "Newsletter for subscribe|unsubscribe")
	formatFlag           = flag.String("format", "table", "Ouput format of subscribers: csv|tsv|table|raw|yaml")
	nameFlag             = flag.String("name", "", "(optional) Name for subscribe")
	logPathFlag          = flag.String("l", "listing-cli.log", "Absolute path to log file")
	stdoutFlag           = flag.Bool("stdout", false, "Log to stdout and to logfile")
	helpFlag             = flag.Bool("help", false, "Print help")
	dryRunFlag           = flag.Bool("dry-run", false, "Simulate selected action")
	noUnconfirmedFlag    = flag.Bool("no-unconfirmed", false, "Do not export unconfirmed emails")
	noConfirmedFlag      = flag.Bool("no-confirmed", false, "Do not export confirmed emails")
	noUnsubscribedFlag   = flag.Bool("no-unsubscribed", false, "Do not export unsubscribed emails")
	ignoreComplaintsFlag = flag.Bool("ignore-complaints", false, "Ignore bounces and complaints for export")
)

const (
	appName         = "listing-cli"
	modeSubscribe   = "subscribe"
	modeUnsubscribe = "unsubscribe"
	modeExport      = "export"
	modeImport      = "import"
	modeDelete      = "delete"
	modeFilter      = "filter"
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
		printer:          NewPrinter(),
		url:              *urlFlag,
		authToken:        *authTokenFlag,
		secret:           *secretFlag,
		complaints:       make(map[string]bool),
		dryRun:           *dryRunFlag,
		noUnconfirmed:    *noUnconfirmedFlag,
		noConfirmed:      *noConfirmedFlag,
		noUnsubscribed:   *noUnsubscribedFlag,
		ignoreComplaints: *ignoreComplaintsFlag,
	}

	switch *modeFlag {
	case modeExport:
		{
			err = client.export(*newsletterFlag)
		}
	case modeFilter:
		{
			bytes, _ := ioutil.ReadAll(os.Stdin)
			err = client.filter(bytes)
		}
	case modeSubscribe:
		{
			err = client.subscribe(*emailFlag, *newsletterFlag, *nameFlag)
		}
	case modeUnsubscribe:
		{
			err = client.unsubscribe(*emailFlag, *newsletterFlag)
		}
	case modeImport:
		{
			bytes, _ := ioutil.ReadAll(os.Stdin)
			err = client.importSubscribers(bytes)
		}
	case modeDelete:
		{
			bytes, _ := ioutil.ReadAll(os.Stdin)
			err = client.deleteSubscribers(bytes)
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

func parseFlags() (err error) {
	flag.Parse()

	switch *modeFlag {
	case "":
		err = errors.New("Mode is required")
	case modeDelete, modeExport, modeImport, modeSubscribe, modeUnsubscribe, modeFilter:
		err = nil
	default:
		err = fmt.Errorf("Mode %v is not supported", *modeFlag)
	}
	if err != nil {
		return
	}

	if *modeFlag != modeFilter {
		switch *urlFlag {
		case "":
			err = errors.New("Url is required")
		default:
			if _, e := url.Parse(*urlFlag); e != nil {
				err = fmt.Errorf("Failed to parse url. err=%v", e)
			}
		}
	}
	if err != nil {
		return
	}

	switch *modeFlag {
	case modeExport, modeUnsubscribe, modeFilter:
		if *secretFlag == "" {
			err = errors.New("Secret flag is required")
		}
	}
	if err != nil {
		return
	}

	switch *modeFlag {
	case modeExport, modeImport, modeDelete:
		if *authTokenFlag == "" {
			err = errors.New("Auth token is required")
		}
	}
	return
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
	case "json":
		return NewJsonPrinter(*secretFlag)
	case "yaml":
		return NewYamlPrinter(*secretFlag)
	default:
		return NewTablePrinter(*secretFlag)
	}
}
