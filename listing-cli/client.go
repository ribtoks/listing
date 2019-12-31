package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ribtoks/listing/pkg/common"
)

type listingClient struct {
	client           *http.Client
	printer          Printer
	url              string
	authToken        string
	secret           string
	complaints       map[string]bool
	dryRun           bool
	noUnconfirmed    bool
	noUnsubscribed   bool
	ignoreComplaints bool
}

func (c *listingClient) endpoint(e string) string {
	baseURL := c.url
	if strings.HasSuffix(baseURL, "/") {
		baseURL = strings.TrimRight(baseURL, "/")
	}
	baseURL = baseURL + e
	return baseURL
}

func (c *listingClient) subscribersURL(newsletter string) (string, error) {
	u, err := url.Parse(c.endpoint(common.SubscribersEndpoint))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set(common.ParamNewsletter, newsletter)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *listingClient) subscribeURL() (string, error) {
	u, err := url.Parse(c.endpoint(common.SubscribeEndpoint))
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *listingClient) complaintsURL() (string, error) {
	u, err := url.Parse(c.endpoint(common.ComplaintsEndpoint))
	if err != nil {
		return "", err
	}
	return u.String(), nil
}
