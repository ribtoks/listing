package main

import (
	"net/url"
	"strings"

	"github.com/ribtoks/listing/pkg/common"
)

type listingClient struct {
	printer   Printer
	url       string
	authToken string
	secret    string
}

func (c *listingClient) subscribersQuery(newsletter string) (string, error) {
	baseURL := c.url
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}
	baseURL = baseURL + common.SubscribersEndpoint
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set(common.ParamNewsletter, newsletter)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
