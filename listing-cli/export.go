package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ribtoks/listing/pkg/common"
)

var (
	errInvalidNewsletter = errors.New("Invalid newsletter parameter")
)

func (c *listingClient) fetchSubscribers(url string) ([]*common.Subscriber, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("any", c.authToken)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

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
