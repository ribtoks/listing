package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ribtoks/listing/pkg/common"
)

func (c *listingClient) prepareDeletePayload(data []byte) ([]byte, error) {
	subscribers, err := c.parseSubscribers(data)
	if err != nil {
		return nil, err
	}
	keys := make([]*common.SubscriberKey, 0)
	for _, s := range subscribers {
		keys = append(keys, &common.SubscriberKey{
			Email:      s.Email,
			Newsletter: s.Newsletter,
		})
	}
	return json.Marshal(keys)
}

func (c *listingClient) sendDeleteRequest(endpoint string, payload []byte) error {
	req, err := http.NewRequest("DELETE", endpoint, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("any", c.authToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))

	}
	return nil
}

func (c *listingClient) deleteSubscribers(data []byte) error {
	endpoint, err := c.importURL()
	if err != nil {
		return err
	}
	payload, err := c.prepareDeletePayload(data)
	if err != nil {
		return err
	}
	log.Printf("About to send delete request. bytes=%v", len(payload))
	if c.dryRun {
		log.Println("Dry run mode. Exiting...")
		return nil
	}
	return c.sendDeleteRequest(endpoint, payload)
}
