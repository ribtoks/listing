package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/go-gomail/gomail"
	"github.com/ribtoks/listing/pkg/common"
)

var (
	smtpServerFlag   = flag.String("url", "", "SMTP server url")
	smtpUsernameFlag = flag.String("user", "", "SMTP username flag")
	smtpPassFlag     = flag.String("pass", "", "SMTP password flag")
	subjectFlag      = flag.String("subject", "", "Html campaign subject")
	fromEmailFlag    = flag.String("from-email", "", "Sender address")
	fromNameFlag     = flag.String("from-name", "", "Sender name")
	htmlTemplateFlag = flag.String("html-template", "", "Path to html email template")
	txtTemplateFlag  = flag.String("txt-template", "", "Path to text email template")
	paramsFlag       = flag.String("params", "params.json", "Path to file with common params")
	workersFlag      = flag.Int("workers", 2, "Number of workers to send emails")
	listFlag         = flag.String("list", "list.json", "Path to file with email list")
	rateFlag         = flag.Int("rate", 25, "Emails per second sending rate")
	dryRunFlag       = flag.Bool("dry-run", false, "Simulate selected action")
	outFlag          = flag.String("out", "./", "Path to directory for dry run results")
	helpFlag         = flag.Bool("help", false, "Print help")
	logPathFlag      = flag.String("l", "listing-send.log", "Absolute path to log file")
	stdoutFlag       = flag.Bool("stdout", false, "Log to stdout and to logfile")
)

const (
	appName           = "listing-send"
	smtpRetryAttempts = 3
	smtpRetrySleep    = 1 * time.Second
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

	htmlTemplate, err := template.ParseFiles(*htmlTemplateFlag)
	if err != nil {
		log.Fatal(err)
	}

	textTemplate, err := template.ParseFiles(*txtTemplateFlag)
	if err != nil {
		log.Fatal(err)
	}

	params, err := readParams(*paramsFlag)
	if err != nil {
		log.Fatal(err)
	}

	subscribers, err := readSubscribers(*listFlag)
	if err != nil {
		log.Fatal(err)
	}

	c := &campaign{
		htmlTemplate: htmlTemplate,
		textTemplate: textTemplate,
		params:       params,
		subscribers:  subscribers,
		subject:      *subjectFlag,
		fromEmail:    *fromEmailFlag,
		fromName:     *fromNameFlag,
		rate:         *rateFlag,
		dryRun:       *dryRunFlag,
		messages:     make(chan *gomail.Message, 10),
		waiter:       &sync.WaitGroup{},
		workersCount: *workersFlag,
	}

	c.send()
}

func readSubscribers(filepath string) ([]*common.SubscriberEx, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.DisallowUnknownFields()

	var subscribers []*common.SubscriberEx
	err = dec.Decode(&subscribers)
	if err != nil {
		return nil, err
	}
	log.Printf("Parsed subscribers. count=%v", len(subscribers))

	return subscribers, nil
}

func readParams(filepath string) (map[string]interface{}, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	params := make(map[string]interface{})
	err = json.Unmarshal(data, &params)
	return params, err
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
	return nil
}

func createSender() (gomail.SendCloser, error) {
	if *dryRunFlag {
		return &dryRunSender{out: *outFlag}, nil
	}

	dialer, err := smtpDialer(*smtpServerFlag, *smtpUsernameFlag, *smtpPassFlag)
	if err != nil {
		return nil, err
	}

	var sender gomail.SendCloser
	for i := 0; i < smtpRetryAttempts; i++ {
		sender, err = dialer.Dial()
		if err == nil {
			log.Printf("Dialed to SMTP. server=%v", *smtpServerFlag)
			break
		} else {
			log.Printf("Failed to dial SMTP. err=%v attempt=%v", err, i)
			log.Printf("Sleeping before retry. interval=%v", smtpRetrySleep)
			time.Sleep(smtpRetrySleep)
		}
	}
	return sender, nil
}
