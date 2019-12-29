package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

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

func (c *listingClient) export(newsletter string) error {
	if newsletter == "" {
		return errInvalidNewsletter
	}
	endpoint, err := c.subscribersURL(newsletter)
	if err != nil {
		return err
	}

	ss, err := c.fetchSubscribers(endpoint)
	if err != nil {
		return err
	}
	for _, s := range ss {
		c.printer.Append(s)
	}
	c.printer.Render()
	return nil
}
