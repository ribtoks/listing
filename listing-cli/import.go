package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/ribtoks/listing/pkg/common"
)

func (c *listingClient) parseSubscribers(data []byte) ([]*common.Subscriber, error) {
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.DisallowUnknownFields()

	var subscribers []*common.Subscriber
	err := dec.Decode(&subscribers)
	if err != nil {
		return nil, err
	}
	return subscribers, nil
}

func (c *listingClient) preparePayload(data []byte) ([]byte, error) {
	subscribers, err := c.parseSubscribers(data)
	if err != nil {
		return nil, err
	}
	return json.Marshal(subscribers)
}

func (c *listingClient) sendImportRequest(endpoint string, payload []byte) error {
	req, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(payload))
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

func (c *listingClient) importData(data []byte) error {
	endpoint, err := c.importURL()
	if err != nil {
		return err
	}
	payload, err := c.preparePayload(data)
	if err != nil {
		return err
	}

	return c.sendImportRequest(endpoint, payload)
}
