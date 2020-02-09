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
	fromEmail    string
	fromName     string
	rate         int
	workersCount int
	dryRun       bool
	waiter       *sync.WaitGroup
	messages     chan *gomail.Message
}

const xMailer = "listing/0.1 (https://github.com/ribtoks/listing)"

func (c *campaign) send() {
	log.Printf("Starting to send messages. from=%v dry-run=%v", c.fromEmail, c.dryRun)
	c.waiter.Add(1)
	go c.generateMessages()
	log.Printf("Starting workers. count=%v", c.workersCount)
	for i := 0; i < c.workersCount; i++ {
		go c.sendMessages(i)
	}
	log.Println("Waiting for workers. count=%v", c.workersCount)
	c.waiter.Wait()
	close(c.messages)
	log.Println("Finished sending messages")
}

func (c *campaign) generateMessages() {
	defer c.waiter.Done()
	rate := time.Second / time.Duration(c.rate)
	throttle := time.Tick(rate)

	for _, s := range c.subscribers {
		m := gomail.NewMessage()
		if err := c.renderMessage(m, s); err != nil {
			log.Printf("Failed to render message. err=%s", err)
			return
		}
		<-throttle // rate limit
		c.waiter.Add(1)
		c.messages <- m
	}
}

func (c *campaign) renderMessage(m *gomail.Message, s *common.SubscriberEx) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	recepient := make(map[string]interface{})
	err = json.Unmarshal(data, &recepient)
	if err != nil {
		return err
	}
	ctx := make(map[string]interface{})
	ctx["Params"] = c.params
	ctx["Recepient"] = recepient

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
	m.SetAddressHeader("From", c.fromEmail, c.fromName)
	m.SetHeader("Subject", c.subject)
	m.SetHeader("X-Mailer", xMailer)
	m.SetBody("text/plain", textBodyTpl.String())
	m.AddAlternative("text/html", htmlBodyTpl.String())
	log.Printf("Rendered email message. recepient=%v", s.Email)
	return nil
}

func (c *campaign) sendMessages(id int) {
	log.Printf("Started sending messages worker. id=%v", id)
	sender, err := createSender()
	if err != nil {
		log.Fatal(err)
	}
	for m := range c.messages {
		if err := gomail.Send(sender, m); err != nil {
			log.Printf("Error sending message. err=%s id=%v to=%v", err, id, m.GetHeader("To"))
			sender.Close()
			sender, err = createSender()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Printf("Sent email. id=%v to=%v", id, m.GetHeader("To"))
		}
		c.waiter.Done()
	}
	sender.Close()
}
