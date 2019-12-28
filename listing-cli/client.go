package main

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ribtoks/listing/pkg/common"
)

type listingClient struct {
	client    *http.Client
	printer   Printer
	url       string
	authToken string
	secret    string
}

func (c *listingClient) subscribersURL(newsletter string) (string, error) {
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
