package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ribtoks/listing/pkg/common"
)

var (
	emptyComplaints []*common.SesNotification
)

func (c *listingClient) fetchComplaints(url string) ([]*common.SesNotification, error) {
	log.Printf("About to fetch complaints. url=%v", url)
	if c.dryRun {
		log.Printf("Dry run mode. Exiting...")
		return emptyComplaints, nil
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

	log.Printf("Received complaints response. status=%v", resp.StatusCode)

	defer resp.Body.Close()
	ss := make([]*common.SesNotification, 0)
	err = json.NewDecoder(resp.Body).Decode(&ss)
	return ss, nil
}

func (c *listingClient) updateComplaints() error {
	endpoint, err := c.complaintsURL()
	if err != nil {
		return err
	}

	complaints, err := c.fetchComplaints(endpoint)
	if err != nil {
		return err
	}

	for _, ct := range complaints {
		if ct.Notification == common.SoftBounceType {
			continue
		}

		c.complaints[ct.Email] = true
	}

	return nil
}
