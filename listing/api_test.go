package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ribtoks/listing/pkg/common"
	"github.com/ribtoks/listing/pkg/db"
)

const (
	subscribeEndpoint   = "/subscribe"
	subscribersEndpoint = "/subscribers"
	complaintsEndpoint  = "/complaints"
	unsubscribeEndpoint = "/unsubscribe"
	confirmEndpoint     = "/confirm"
	secret              = "secret123"
	apiToken            = "qwerty123456"
	testName            = "Foo Bar"
	testEmail           = "foo@bar.com"
	testNewsletter      = "testnewsletter"
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

func NewTestResource(router *http.ServeMux, subscribers common.SubscribersStore, notifications common.NotificationsStore) *NewsletterResource {
	newsletters := &NewsletterResource{
		subscribers:   subscribers,
		notifications: notifications,
		secret:        secret,
		apiToken:      apiToken,
		newsletters:   make(map[string]bool),
		mailer:        &DevNullMailer{},
	}
	return newsletters
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

func TestGetSubscribeMethodIsNotSupported(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

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
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

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
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

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
	newsletter := "foo"
	store := NewSubscribersStore()
	nr := NewTestResource(srv, store, NewNotificationsStore())
	nr.addNewsletters([]string{newsletter})
	nr.setup(srv)

	data := url.Values{}
	data.Set("newsletter", newsletter)
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

	if len(store.items) != 1 {
		t.Errorf("Wrong number of items in the store: %v", len(store.items))
	}
}

func TestConfirmSubscribe(t *testing.T) {
	srv := http.NewServeMux()

	store := NewSubscribersStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(srv, store, NewNotificationsStore())
	nr.addNewsletters([]string{testNewsletter})
	nr.setup(srv)

	data := url.Values{}
	data.Set("newsletter", testNewsletter)
	data.Set("token", common.Sign(secret, testEmail))

	req, err := http.NewRequest("GET", confirmEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("newsletter", testNewsletter)
	q.Add("token", common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusFound {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}

	i := store.items[store.key(testNewsletter, testEmail)]
	if i.ConfirmedAt.Time().Sub(i.CreatedAt.Time()) < 0 {
		t.Errorf("Confirm time not updated")
	}
}

func TestGetSubscribersUnauthorized(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("GET", subscribersEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestGetSubscribersWithWrongPassword(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("GET", subscribersEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.SetBasicAuth("any username", "wrong password")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestGetSubscribersWithoutParam(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("GET", subscribersEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.SetBasicAuth("any username", apiToken)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestGetSubscribersWrongNewsletter(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("GET", subscribersEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add("newsletter", "test")
	req.URL.RawQuery = q.Encode()
	req.SetBasicAuth("any username", apiToken)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func TestGetSubscribersOK(t *testing.T) {
	srv := http.NewServeMux()

	store := NewSubscribersStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(srv, store, NewNotificationsStore())
	nr.setup(srv)
	nr.addNewsletters([]string{testNewsletter})

	req, err := http.NewRequest("GET", subscribersEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add("newsletter", testNewsletter)
	req.URL.RawQuery = q.Encode()

	req.SetBasicAuth("any username", apiToken)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	ss := make([]*common.Subscriber, 0)
	err = json.Unmarshal(body, &ss)
	if err != nil {
		t.Fatal(err)
	}

	if len(ss) != 1 {
		t.Errorf("Wrong number of items in response: %v", len(ss))
	}

	if ss[0].Email != testEmail {
		t.Errorf("Wrong data received: %v", body)
	}
}

func TestUnsubscribeWrongMethod(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("POST", unsubscribeEndpoint, nil)
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

func TestUnsubscribeWithoutNewsletter(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("GET", unsubscribeEndpoint, nil)
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

func TestUnsubscribeWithoutToken(t *testing.T) {
	srv := http.NewServeMux()

	store := NewSubscribersStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(srv, store, NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("GET", unsubscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("newsletter", "random value")
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	if len(store.items) != 1 {
		t.Errorf("Wrong number of subscribers: %v", len(store.items))
	}
}

func TestUnsubscribeWithBadToken(t *testing.T) {
	srv := http.NewServeMux()

	store := NewSubscribersStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(srv, store, NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("GET", unsubscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add("newsletter", "random value")
	q.Add("token", "abcde")
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	if len(store.items) != 1 {
		t.Errorf("Wrong number of subscribers: %v", len(store.items))
	}
}

func TestUnsubscribe(t *testing.T) {
	srv := http.NewServeMux()

	store := NewSubscribersStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestResource(srv, store, NewNotificationsStore())
	nr.addNewsletters([]string{testNewsletter})
	nr.setup(srv)

	req, err := http.NewRequest("GET", unsubscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add("newsletter", testNewsletter)
	q.Add("token", common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	if len(store.items) != 1 {
		t.Errorf("Wrong number of subscribers left: %d", len(store.items))
	}

	i := store.items[store.key(testNewsletter, testEmail)]
	if i.UnsubscribedAt.Time().Sub(i.CreatedAt.Time()) < 0 {
		t.Errorf("Unsubscribe time not updated")
	}
}

func TestPutSubscribersUnauthorized(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("PUT", subscribersEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestPutSubscribersWrongNewsletter(t *testing.T) {
	newsletter := "TestNewsletter"

	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

	var subscribers []*common.Subscriber
	for i := 0; i < 10; i++ {
		subscribers = append(subscribers, &common.Subscriber{
			Newsletter:     newsletter,
			Email:          fmt.Sprintf("foo%v@bar.com", i),
			CreatedAt:      common.JsonTimeNow(),
			UnsubscribedAt: incorrectTime,
			ConfirmedAt:    incorrectTime,
		})
	}
	data, err := json.Marshal(subscribers)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PUT", subscribersEndpoint, bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("any username", apiToken)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func TestPutSubscribers(t *testing.T) {
	newsletter := "TestNewsletter"

	srv := http.NewServeMux()
	store := NewSubscribersStore()
	nr := NewTestResource(srv, store, NewNotificationsStore())
	nr.setup(srv)
	nr.addNewsletters([]string{newsletter})

	expectedEmails := make(map[string]bool)
	var subscribers []*common.Subscriber
	for i := 0; i < 10; i++ {
		subscribers = append(subscribers, &common.Subscriber{
			Newsletter:     newsletter,
			Email:          fmt.Sprintf("foo%v@bar.com", i),
			CreatedAt:      common.JsonTimeNow(),
			UnsubscribedAt: incorrectTime,
			ConfirmedAt:    incorrectTime,
		})
		expectedEmails[subscribers[i].Email] = true
	}
	data, err := json.Marshal(subscribers)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("PUT", subscribersEndpoint, bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("any username", apiToken)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}

	for k, _ := range expectedEmails {
		if !store.contains(newsletter, k) {
			t.Errorf("Email not imported: %v", k)
		}
	}
}

func TestGetComplaintsUnauthorized(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestResource(srv, NewSubscribersStore(), NewNotificationsStore())
	nr.setup(srv)

	req, err := http.NewRequest("GET", complaintsEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestGetComplaintsOK(t *testing.T) {
	srv := http.NewServeMux()
	store := NewNotificationsStore()
	store.AddBounce(testEmail, "from@email.com", false /*is transient*/)
	store.AddComplaint(testEmail, "from@email.com")
	nr := NewTestResource(srv, NewSubscribersStore(), store)
	nr.setup(srv)

	req, err := http.NewRequest("GET", complaintsEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.SetBasicAuth("any username", apiToken)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	ss := make([]*common.SesNotification, 0)
	err = json.Unmarshal(body, &ss)
	if err != nil {
		t.Fatal(err)
	}

	if len(ss) != 2 {
		t.Errorf("Wrong number of items in response: %v", len(ss))
	}
}
