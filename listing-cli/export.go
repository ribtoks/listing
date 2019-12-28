package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/ribtoks/listing/pkg/common"
)

var (
	errInvalidNewsletter = errors.New("Invalid newsletter parameter")
)

func (c *listingClient) fetchSubscribers(newsletter string) ([]*common.Subscriber, error) {
	if newsletter == "" {
		return nil, errInvalidNewsletter
	}
	endpoint, err := c.subscribersQuery(newsletter)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("any", c.authToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ss := make([]*common.Subscriber, 0)
	err = json.Unmarshal(body, &ss)
	if err != nil {
		return nil, err
	}

	return ss, nil
}

func (c *listingClient) export(newsletter string) error {
	ss, err := c.fetchSubscribers(newsletter)
	if err != nil {
		return err
	}
	for _, s := range ss {
		c.printer.Append(s)
	}
	c.printer.Render()
	return nil
}
