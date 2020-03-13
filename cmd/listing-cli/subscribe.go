package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ribtoks/listing/pkg/common"
)

var (
	errMissingNewsletter = errors.New("Newsletter parameter is empty")
	errMissingEmail      = errors.New("Email parameter is empty")
)

func (c *listingClient) sendSubscribeRequest(email, newsletter, name, endpoint string) error {
	log.Printf("About to send subscribe request. email=%v newsletter=%v name=%v url=%v", email, newsletter, name, endpoint)
	if c.dryRun {
		log.Println("Dry run mode. Exiting...")
		return nil
	}
	data := url.Values{}
	data.Set(common.ParamNewsletter, newsletter)
	data.Set(common.ParamEmail, email)
	if name != "" {
		data.Set(common.ParamName, name)
	}
	encoded := data.Encode()
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(encoded))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(encoded)))

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	log.Printf("Received subscribe response. status=%v", resp.StatusCode)
	if resp.StatusCode != http.StatusFound {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
	return nil
}

func (c *listingClient) subscribe(email, newsletter, name string) error {
	if email == "" {
		return errMissingEmail
	}

	if newsletter == "" {
		return errMissingNewsletter
	}

	url, err := c.subscribeURL()
	if err != nil {
		return err
	}

	return c.sendSubscribeRequest(email, newsletter, name, url)
}
