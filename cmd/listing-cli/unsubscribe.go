package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ribtoks/listing/pkg/common"
)

func (c *listingClient) sendUnsubscribeRequest(email, newsletter, endpoint string) error {
	log.Printf("About to send unsubscribe request. email=%v newsletter=%v url=%v", email, newsletter, endpoint)
	if c.dryRun {
		log.Println("Dry run mode. Exiting...")
		return nil
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	q := req.URL.Query()
	q.Add(common.ParamNewsletter, newsletter)
	q.Add(common.ParamToken, common.Sign(c.secret, email))
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	log.Printf("Received unsubscribe response. status=%v", resp.StatusCode)
	if resp.StatusCode != http.StatusFound {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
	return nil
}

func (c *listingClient) unsubscribe(email, newsletter string) error {
	if email == "" {
		return errMissingEmail
	}

	if newsletter == "" {
		return errMissingNewsletter
	}

	url, err := c.unsubscribeURL()
	if err != nil {
		return err
	}

	return c.sendUnsubscribeRequest(email, newsletter, url)
}
