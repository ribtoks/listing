package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/ribtoks/listing/pkg/common"
)

var (
	errInvalidNewsletter = errors.New("Invalid newsletter parameter")
	emptySubscribers     []*common.Subscriber
)

func (c *listingClient) fetchSubscribers(url string) ([]*common.Subscriber, error) {
	log.Printf("About to fetch subscribers. url=%v", url)
	if c.dryRun {
		log.Println("Dry run mode. Exiting...")
		return emptySubscribers, nil
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("any", c.authToken)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	log.Printf("Received subscribers response. status=%v", resp.StatusCode)

	defer resp.Body.Close()
	ss := make([]*common.Subscriber, 0)
	err = json.NewDecoder(resp.Body).Decode(&ss)
	return ss, nil
}

func (c *listingClient) isSubscriberOK(s *common.Subscriber) bool {
	if c.noUnconfirmed && !s.Confirmed() {
		log.Printf("Skipping unconfirmed subscriber. created_at=%v confirmed_at=%v confirmed=%v", s.CreatedAt, s.ConfirmedAt, s.Confirmed())
		return false
	}

	if c.noUnsubscribed && s.Unsubscribed() {
		log.Printf("Skipping unsubscribed subscriber. created_at=%v unsubscribed_at=%v unsubscribed=%v", s.CreatedAt, s.UnsubscribedAt, s.Unsubscribed())
		return false
	}

	if c.noConfirmed && s.Confirmed() {
		log.Printf("Skipping confirmed subscriber. created_at=%v confirmed_at=%v confirmed=%v", s.CreatedAt, s.ConfirmedAt, s.Confirmed())
		return false
	}

	if _, ok := c.complaints[s.Email]; ok {
		log.Printf("Skipping bounced or complained subscriber. email=%v", s.Email)
		return false
	}

	return true
}

func (c *listingClient) export(newsletter string) error {
	if newsletter == "" {
		return errInvalidNewsletter
	}
	endpoint, err := c.subscribersURL(newsletter)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	if !c.ignoreComplaints {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := c.updateComplaints()
			if err != nil {
				log.Printf("Failed to update complaints. err=%v", err)
			}
		}()
	}

	ss, err := c.fetchSubscribers(endpoint)
	if err != nil {
		return err
	}

	wg.Wait()

	skipped := 0
	for _, s := range ss {
		if c.isSubscriberOK(s) {
			c.printer.Append(s)
		} else {
			skipped += 1
		}
	}
	c.printer.Render()
	log.Printf("Exported subscribers. count=%v skipped=%v", len(ss), skipped)
	return nil
}
