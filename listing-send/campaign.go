package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"sync"
	"time"

	"github.com/go-gomail/gomail"
	"github.com/ribtoks/listing/pkg/common"
)

type campaign struct {
	htmlTemplate *template.Template
	textTemplate *template.Template
	params       map[string]interface{}
	subscribers  []*common.SubscriberEx
	subject      string
	from         string
	rate         int
	dryRun       bool
	waiter       *sync.WaitGroup
	messages     chan *gomail.Message
}

const xMailer = "listing/0.1 (https://github.com/ribtoks/listing)"

func (c *campaign) send(sender gomail.SendCloser) {
	go c.generateMessages()
	c.sendMessages(sender)
}

func (c *campaign) generateMessages() {
	rate := time.Second / time.Duration(c.rate)
	throttle := time.Tick(rate)

	for _, s := range c.subscribers {
		m := gomail.NewMessage()
		if err := c.renderMessage(m, s); err != nil {
			log.Printf("Failed to render message. err=%s", err)
			return
		}
		<-throttle // rate limit
		c.messages <- m
	}
	close(c.messages)
}

func (c *campaign) renderMessage(m *gomail.Message, s *common.SubscriberEx) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	ctx := make(map[string]interface{})
	err = json.Unmarshal(data, &ctx)
	if err != nil {
		return err
	}
	ctx["Params"] = c.params

	var htmlBodyTpl bytes.Buffer
	if err := c.htmlTemplate.Execute(&htmlBodyTpl, ctx); err != nil {
		return err
	}

	var textBodyTpl bytes.Buffer
	if err := c.textTemplate.Execute(&textBodyTpl, ctx); err != nil {
		return err
	}

	m.Reset() // Return to NewMessage state
	m.SetAddressHeader("To", s.Email, s.Name)
	m.SetHeader("Subject", c.subject)
	m.SetHeader("From", c.from)
	m.SetHeader("X-Mailer", xMailer)
	m.SetBody("text/plain", textBodyTpl.String())
	m.AddAlternative("text/html", htmlBodyTpl.String())
	return nil
}

func (c *campaign) sendMessages(sender gomail.SendCloser) {
	for m := range c.messages {
		if err := gomail.Send(sender, m); err != nil {
			log.Printf("Error sending message. err=%s", err)
		}
	}
}
