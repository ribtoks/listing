package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"

	"github.com/go-gomail/gomail"
)

func smtpDialer(smtpUrl, user, pass string) (*gomail.Dialer, error) {
	surl, err := url.Parse(smtpUrl)
	if err != nil {
		return nil, err
	}

	// Port
	var port int
	if i, err := strconv.Atoi(surl.Port()); err == nil {
		port = i
	} else if surl.Scheme == "smtp" {
		port = 25
	} else {
		port = 465
	}

	d := gomail.NewPlainDialer(surl.Hostname(), port, user, pass)
	d.SSL = (surl.Scheme == "smtps")
	return d, nil
}

type dryRunSender struct {
	out string
}

func (s *dryRunSender) Send(from string, to []string, msg io.WriterTo) error {
	var buf bytes.Buffer
	msg.WriteTo(&buf)
	ioutil.WriteFile(s.out+to[0], buf.Bytes(), 0644)
	return nil
}

func (s *dryRunSender) Close() error {
	return nil
}
