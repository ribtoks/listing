package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	subscribeEndpoint = "/subscribe"
	secret            = "secret123"
	apiToken          = "qwerty123456"
)

type MapStore struct {
	items map[string]*Subscriber
}

var _ Store = (*MapStore)(nil)

func (s *MapStore) key(newsletter, email string) string {
	return newsletter + email
}

func (s *MapStore) AddSubscriber(newsletter, email string) error {
	key := s.key(newsletter, email)
	if _, ok := s.items[key]; ok {
		return errors.New("Subscriber already exists")
	}

	s.items[key] = &Subscriber{
		Newsletter:   newsletter,
		Email:        email,
		CreatedAt:    time.Now(),
		ConfirmedAt:  time.Unix(1, 1),
		ComplainedAt: time.Unix(1, 1),
		BouncedAt:    time.Unix(1, 1),
	}
	return nil
}

func (s *MapStore) RemoveSubscriber(newsletter, email string) error {
	key := s.key(newsletter, email)
	if _, ok := s.items[key]; !ok {
		return errors.New("Subscriber does not exist")
	}
	delete(s.items, key)
	return nil
}

func (s *MapStore) GetSubscribers(newsletter string) (subscribers []*Subscriber, err error) {
	for _, value := range s.items {
		subscribers = append(subscribers, value)
	}
	return subscribers, nil
}

func NewTestResource(router *http.ServeMux) *NewsletterResource {
	newsletters := &NewsletterResource{
		store: &MapStore{
			items: make(map[string]*Subscriber),
		},
		secret:   secret,
		apiToken: apiToken,
	}
	return newsletters
}

func TestGetSubscribeMethodIsNotSupported(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv)
	nr.Setup(srv)

	req, err := http.NewRequest("GET", subscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestSubscribeWithoutParams(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv)
	nr.Setup(srv)

	req, err := http.NewRequest("POST", subscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestSubscribeWithBadEmail(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv)
	nr.Setup(srv)

	data := url.Values{}
	data.Set("newsletter", "foo")
	data.Set("email", "bar")

	req, err := http.NewRequest("POST", subscribeEndpoint, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestSubscribe(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv)
	nr.Setup(srv)

	data := url.Values{}
	data.Set("newsletter", "foo")
	data.Set("email", "bar@foo.com")

	req, err := http.NewRequest("POST", subscribeEndpoint, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}
