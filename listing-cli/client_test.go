package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ribtoks/listing/pkg/api"
	"github.com/ribtoks/listing/pkg/common"
	"github.com/ribtoks/listing/pkg/db"
)

const (
	secret         = "secret123"
	apiToken       = "qwerty123456"
	testName       = "Foo Bar"
	testEmail      = "foo@bar.com"
	testNewsletter = "testnewsletter"
)

var incorrectTime = common.JSONTime(time.Unix(1, 1))

type DevNullMailer struct{}

func (m *DevNullMailer) SendConfirmation(newsletter, email, name, confirmUrl string) error {
	return nil
}

type SubscribersMapStore struct {
	items map[string]*common.Subscriber
}

var _ common.SubscribersStore = (*SubscribersMapStore)(nil)

func (s *SubscribersMapStore) key(newsletter, email string) string {
	return newsletter + email
}

func (s *SubscribersMapStore) contains(newsletter, email string) bool {
	_, ok := s.items[s.key(newsletter, email)]
	return ok
}

func (s *SubscribersMapStore) AddSubscriber(newsletter, email, name string) error {
	key := s.key(newsletter, email)
	if _, ok := s.items[key]; ok {
		return errors.New("Subscriber already exists")
	}

	s.items[key] = &common.Subscriber{
		Newsletter:     newsletter,
		Email:          email,
		CreatedAt:      common.JsonTimeNow(),
		ConfirmedAt:    incorrectTime,
		UnsubscribedAt: incorrectTime,
	}
	return nil
}

func (s *SubscribersMapStore) RemoveSubscriber(newsletter, email string) error {
	key := s.key(newsletter, email)
	if i, ok := s.items[key]; ok {
		i.UnsubscribedAt = common.JsonTimeNow()
		return nil
	}
	return errors.New("Subscriber does not exist")
}

func (s *SubscribersMapStore) DeleteSubscribers(keys []*common.SubscriberKey) error {
	for _, k := range keys {
		key := s.key(k.Newsletter, k.Email)
		delete(s.items, key)
	}
	return nil
}

func (s *SubscribersMapStore) Subscribers(newsletter string) (subscribers []*common.Subscriber, err error) {
	for key, value := range s.items {
		if strings.HasPrefix(key, newsletter) {
			subscribers = append(subscribers, value)
		}
	}
	return subscribers, nil
}

func (s *SubscribersMapStore) AddSubscribers(subscribers []*common.Subscriber) error {
	for _, i := range subscribers {
		s.items[s.key(i.Newsletter, i.Email)] = i
	}
	return nil
}

func (s *SubscribersMapStore) ConfirmSubscriber(newsletter, email string) error {
	key := s.key(newsletter, email)
	if i, ok := s.items[key]; ok {
		i.ConfirmedAt = common.JsonTimeNow()
		return nil
	}
	return errors.New("Subscriber does not exist")
}

func NewSubscribersStore() *SubscribersMapStore {
	return &SubscribersMapStore{
		items: make(map[string]*common.Subscriber),
	}
}

func NewNotificationsStore() *NotificationsMapStore {
	return &NotificationsMapStore{
		items: make([]*common.SesNotification, 0),
	}
}

type NotificationsMapStore struct {
	items []*common.SesNotification
}

var _ common.NotificationsStore = (*NotificationsMapStore)(nil)

func (s *NotificationsMapStore) AddBounce(email, from string, isTransient bool) error {
	t := db.SoftBounceType
	if !isTransient {
		t = db.HardBounceType
	}
	s.items = append(s.items, &common.SesNotification{
		Email:        email,
		ReceivedAt:   common.JsonTimeNow(),
		Notification: t,
		From:         from,
	})
	return nil
}

func (s *NotificationsMapStore) AddComplaint(email, from string) error {
	s.items = append(s.items, &common.SesNotification{
		Email:        email,
		ReceivedAt:   common.JsonTimeNow(),
		Notification: db.ComplaintType,
		From:         from,
	})
	return nil
}

func (s *NotificationsMapStore) Notifications() (notifications []*common.SesNotification, err error) {
	return s.items, nil
}

func NewTestResource(subscribers common.SubscribersStore, notifications common.NotificationsStore) *api.NewsletterResource {
	newsletters := &api.NewsletterResource{
		Subscribers:   subscribers,
		Notifications: notifications,
		Secret:        secret,
		ApiToken:      apiToken,
		Newsletters:   make(map[string]bool),
		Mailer:        &DevNullMailer{},
	}
	return newsletters
}

func testingHttpClient(handler http.HandlerFunc) (*http.Client, func()) {
	s := httptest.NewServer(handler)

	cli := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, network, _ string) (net.Conn, error) {
				return net.Dial(network, s.Listener.Addr().String())
			},
		},
	}

	return cli, s.Close
}

type RawTestPrinter struct {
	subscribers []*common.Subscriber
}

func NewRawTestPrinter() *RawPrinter {
	rp := &RawPrinter{
		subscribers: make([]*common.Subscriber, 0),
	}
	return rp
}

func (rp *RawTestPrinter) Append(s *common.Subscriber) {
	rp.subscribers = append(rp.subscribers, s)
}

func (rp *RawTestPrinter) Render() error {
	return nil
}

func NewTestClient(resource *api.NewsletterResource, p Printer, mux *http.ServeMux) (*httptest.Server, *listingClient) {
	server := httptest.NewServer(mux)
	resource.Setup(mux)

	client := &listingClient{
		client:         &http.Client{},
		printer:        p,
		url:            server.URL,
		authToken:      apiToken,
		secret:         secret,
		dryRun:         false,
		noUnconfirmed:  false,
		noUnsubscribed: false,
	}

	return server, client
}

func TestExportSubscribers(t *testing.T) {
	store := NewSubscribersStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(store, NewNotificationsStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	mux := http.NewServeMux()
	srv, cli := NewTestClient(nr, p, mux)
	defer srv.Close()

	err := cli.export(testNewsletter)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.subscribers) != 1 {
		t.Errorf("Unexpected number of subscribers: %v", len(p.subscribers))
	}
}