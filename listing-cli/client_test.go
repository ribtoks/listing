package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
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

var (
	incorrectTime       = common.JSONTime(time.Unix(1, 1))
	errFromFailingStore = errors.New("Error!")
)

type FailingSubscriberStore struct{}

var _ common.SubscribersStore = (*FailingSubscriberStore)(nil)

func (s *FailingSubscriberStore) AddSubscriber(newsletter, email, name string) error {
	return errFromFailingStore
}

func (s *FailingSubscriberStore) RemoveSubscriber(newsletter, email string) error {
	return errFromFailingStore
}

func (s *FailingSubscriberStore) Subscribers(newsletter string) (subscribers []*common.Subscriber, err error) {
	return nil, errFromFailingStore
}

func (s *FailingSubscriberStore) AddSubscribers(subscribers []*common.Subscriber) error {
	return errFromFailingStore
}

func (s *FailingSubscriberStore) DeleteSubscribers(keys []*common.SubscriberKey) error {
	return errFromFailingStore
}

func (s *FailingSubscriberStore) ConfirmSubscriber(newsletter, email string) error {
	return errFromFailingStore
}

func NewFailingStore() *FailingSubscriberStore {
	return &FailingSubscriberStore{}
}

type DevNullMailer struct{}

func (m *DevNullMailer) SendConfirmation(newsletter, email, name, confirmUrl string) error {
	return nil
}

func alternateConfirm(ss []*common.Subscriber) {
	for i, v := range ss {
		if i%2 == 0 {
			v.ConfirmedAt = common.JSONTime(v.CreatedAt.Time().Add(1 * time.Second))
		}
	}
}

func alternateUnsubscribe(ss []*common.Subscriber) {
	for i, v := range ss {
		if i%2 == 0 {
			v.UnsubscribedAt = common.JSONTime(v.CreatedAt.Time().Add(1 * time.Second))
		}
	}
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

func NewRawTestPrinter() *RawTestPrinter {
	rp := &RawTestPrinter{
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

func NewTestClient(resource *api.NewsletterResource, p Printer) (*httptest.Server, *listingClient) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	resource.Setup(mux)

	client := &listingClient{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		printer:          p,
		url:              server.URL,
		authToken:        apiToken,
		secret:           secret,
		complaints:       make(map[string]bool),
		dryRun:           false,
		noUnconfirmed:    false,
		noUnsubscribed:   false,
		ignoreComplaints: false,
	}

	return server, client
}

func TestExportSubscribedSubscribers(t *testing.T) {
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, "email1@domain.com", testName)
	store.AddSubscriber(testNewsletter, "email2@domain.com", testName)
	ss, _ := store.Subscribers(testNewsletter)
	alternateUnsubscribe(ss)

	nr := NewTestResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	cli.noUnsubscribed = true
	err := cli.export(testNewsletter)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.subscribers) != 1 {
		t.Errorf("Unexpected number of subscribers: %v", len(p.subscribers))
	}
}

func TestExportConfirmedSubscribers(t *testing.T) {
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, "email1@domain.com", testName)
	store.AddSubscriber(testNewsletter, "email2@domain.com", testName)
	ss, _ := store.Subscribers(testNewsletter)
	alternateConfirm(ss)

	nr := NewTestResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	cli.noUnconfirmed = true
	err := cli.export(testNewsletter)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.subscribers) != 1 {
		t.Errorf("Unexpected number of subscribers: %v", len(p.subscribers))
	}
}

func TestExportAllSubscribers(t *testing.T) {
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, "email1@domain.com", testName)
	store.AddSubscriber(testNewsletter, "email2@domain.com", testName)

	nr := NewTestResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	err := cli.export(testNewsletter)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.subscribers) != 2 {
		t.Errorf("Unexpected number of subscribers: %v", len(p.subscribers))
	}
}

func TestSubscribeDryRun(t *testing.T) {
	store := db.NewSubscribersMapStore()
	nr := NewTestResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	cli.dryRun = true
	err := cli.subscribe(testEmail, testNewsletter, testName)
	if err != nil {
		t.Fatal(err)
	}

	if store.Count() != 0 {
		t.Errorf("Unexpected number of subscribers: %v", store.Count())
	}
}

func TestSubscribe(t *testing.T) {
	store := db.NewSubscribersMapStore()
	nr := NewTestResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	err := cli.subscribe(testEmail, testNewsletter, testName)
	if err != nil {
		t.Fatal(err)
	}

	if store.Count() != 1 {
		t.Errorf("Unexpected number of subscribers: %v", store.Count())
	}

	if _, err := store.GetSubscriber(testNewsletter, testEmail); err != nil {
		t.Errorf("Subscriber is not added to the store")
	}
}

func TestUnsubscribeDryRun(t *testing.T) {
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	cli.dryRun = true
	err := cli.unsubscribe(testEmail, testNewsletter)
	if err != nil {
		t.Fatal(err)
	}

	if store.Count() != 1 {
		t.Errorf("Unexpected number of subscribers: %v", store.Count())
	}

	i, _ := store.GetSubscriber(testNewsletter, testEmail)
	if i.Unsubscribed() {
		t.Errorf("Subscriber was unsubscribed")
	}
}

func TestUnsubscribe(t *testing.T) {
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	err := cli.unsubscribe(testEmail, testNewsletter)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(1 * time.Millisecond)

	if store.Count() != 1 {
		t.Errorf("Unexpected number of subscribers: %v", store.Count())
	}

	i, _ := store.GetSubscriber(testNewsletter, testEmail)
	if !i.Unsubscribed() {
		t.Errorf("Subscriber was not unsubscribed. created_at=%v unsubscribed_at=%v", i.CreatedAt, i.UnsubscribedAt)
	}
}

func TestExportEmptyNewsletter(t *testing.T) {
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	err := cli.export("")
	if err == nil {
		t.Fatalf("Managed to export empty newsletter")
	}
}

func TestExportDryRun(t *testing.T) {
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	cli.dryRun = true
	err := cli.export(testNewsletter)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.subscribers) > 0 {
		t.Errorf("Dry run exported data")
	}
}

func TestExportSubscribersWithComplaints(t *testing.T) {
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, "email1@domain.com", testName)
	store.AddSubscriber(testNewsletter, "email2@domain.com", testName)
	store.AddSubscriber(testNewsletter, "email3@domain.com", testName)

	complaints := db.NewNotificationsMapStore()
	complaints.AddBounce("email1@domain.com", "no-reply@newsletter.com", false /*is transient*/)
	complaints.AddBounce("email2@domain.com", "no-reply@newsletter.com", true /*is transient*/)
	complaints.AddComplaint("email3@domain.com", "no-reply@newsletter.com")

	nr := NewTestResource(store, complaints)
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	err := cli.export(testNewsletter)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.subscribers) != 1 {
		t.Errorf("Unexpected number of subscribers: %v", len(p.subscribers))
	}

	if p.subscribers[0].Email != "email2@domain.com" {
		t.Errorf("Wrong subsciber has been exported")
	}
}

func TestExportSubscribersWithoutComplaints(t *testing.T) {
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, "email1@domain.com", testName)
	store.AddSubscriber(testNewsletter, "email2@domain.com", testName)
	store.AddSubscriber(testNewsletter, "email3@domain.com", testName)

	complaints := db.NewNotificationsMapStore()
	complaints.AddBounce("email1@domain.com", "no-reply@newsletter.com", false /*is transient*/)
	complaints.AddBounce("email2@domain.com", "no-reply@newsletter.com", true /*is transient*/)
	complaints.AddComplaint("email3@domain.com", "no-reply@newsletter.com")

	nr := NewTestResource(store, complaints)
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	cli.ignoreComplaints = true
	err := cli.export(testNewsletter)
	if err != nil {
		t.Fatal(err)
	}

	if len(p.subscribers) != 3 {
		t.Errorf("Unexpected number of subscribers: %v", len(p.subscribers))
	}
}

func TestSubscribeErrorStore(t *testing.T) {
	nr := NewTestResource(NewFailingStore(), db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	err := cli.subscribe(testEmail, testNewsletter, testName)
	if err == nil {
		t.Fatal("Subscribed with failing store")
	}
}

func TestUnsubscribeErrorStore(t *testing.T) {
	nr := NewTestResource(NewFailingStore(), db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	err := cli.unsubscribe(testEmail, testNewsletter)
	if err == nil {
		t.Fatal("Unsubscribed with failing store")
	}
}

func TestSubscribeErrors(t *testing.T) {
	nr := NewTestResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	err := cli.subscribe("", testNewsletter, testName)
	if err == nil {
		t.Fatal("Subscribed with empty email")
	}

	err = cli.subscribe(testEmail, "", testName)
	if err == nil {
		t.Fatal("Subscribed with empty newsletter")
	}
}

func TestUnsubscribeErrors(t *testing.T) {
	nr := NewTestResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})

	p := NewRawTestPrinter()
	srv, cli := NewTestClient(nr, p)
	defer srv.Close()

	err := cli.unsubscribe("", testNewsletter)
	if err == nil {
		t.Fatal("Unsubscribed with empty email")
	}

	err = cli.unsubscribe(testEmail, "")
	if err == nil {
		t.Fatal("Unsubscribed with empty newsletter")
	}
}
