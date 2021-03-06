package api

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
	secret         = "secret123"
	apiToken       = "qwerty123456"
	testName       = "Foo Bar"
	testEmail      = "foo@bar.com"
	testNewsletter = "testnewsletter"
	testUrl        = "http://mysupertest.com/location"
)

var incorrectTime = common.JSONTime(time.Unix(1, 1))
var errFromFailingStore = errors.New("Error!")

type DevNullMailer struct{}

func (m *DevNullMailer) SendConfirmation(newsletter, email, name, confirmUrl string) error {
	return nil
}

type FailingSubscriberStore struct {
	failGetSubscriber bool
}

var _ common.SubscribersStore = (*FailingSubscriberStore)(nil)

func (s *FailingSubscriberStore) GetSubscriber(newsletter, email string) (*common.Subscriber, error) {
	if s.failGetSubscriber {
		return nil, errFromFailingStore
	}
	return &common.Subscriber{}, nil
}

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
	return &FailingSubscriberStore{
		failGetSubscriber: true,
	}
}

type FailingNotificationsStore struct{}

func (s *FailingNotificationsStore) AddBounce(email, from string, isTransient bool) error {
	return errFromFailingStore
}

func (s *FailingNotificationsStore) AddComplaint(email, from string) error {
	return errFromFailingStore
}

func (s *FailingNotificationsStore) Notifications() (notifications []*common.SesNotification, err error) {
	return nil, errFromFailingStore
}

func NewTestNewsResource(subscribers common.SubscribersStore, notifications common.NotificationsStore) *NewsletterResource {
	newsletters := &NewsletterResource{
		Subscribers:   subscribers,
		Notifications: notifications,
		Secret:        secret,
		Newsletters:   make(map[string]bool),
		Mailer:        &DevNullMailer{},
	}
	return newsletters
}

func NewTestAdminResource(subscribers common.SubscribersStore, notifications common.NotificationsStore) *AdminResource {
	admins := &AdminResource{
		Subscribers:   subscribers,
		Notifications: notifications,
		APIToken:      apiToken,
		Newsletters:   make(map[string]bool),
	}
	return admins
}

func TestGetSubscribeMethodIsNotSupported(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestNewsResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.SubscribeEndpoint, nil)
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
	nr := NewTestNewsResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("POST", common.SubscribeEndpoint, nil)
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
	nr := NewTestNewsResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	data := url.Values{}
	data.Set(common.ParamNewsletter, "foo")
	data.Set(common.ParamEmail, "bar")

	req, err := http.NewRequest("POST", common.SubscribeEndpoint, strings.NewReader(data.Encode()))
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

func TestSubscribeIncorrectNewsletter(t *testing.T) {
	srv := http.NewServeMux()
	store := db.NewSubscribersMapStore()
	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)

	data := url.Values{}
	data.Set(common.ParamNewsletter, "foo")
	data.Set(common.ParamEmail, "bar@foo.com")

	req, err := http.NewRequest("POST", common.SubscribeEndpoint, strings.NewReader(data.Encode()))
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
	store := db.NewSubscribersMapStore()
	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{newsletter})
	nr.Setup(srv)
	nr.SubscribeRedirectURL = testUrl

	data := url.Values{}
	data.Set(common.ParamNewsletter, newsletter)
	data.Set(common.ParamEmail, "bar@foo.com")

	req, err := http.NewRequest("POST", common.SubscribeEndpoint, strings.NewReader(data.Encode()))
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

	l, err := resp.Location()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != testUrl {
		t.Errorf("Path does not match. expected=%v actual=%v", l.Path, testUrl)
	}

	ss, _ := store.Subscribers(newsletter)
	if len(ss) != 1 {
		t.Errorf("Wrong number of items in the store: %v", len(ss))
	}
}

func TestSubscribeFailingStore(t *testing.T) {
	srv := http.NewServeMux()
	newsletter := "foo"
	nr := NewTestNewsResource(NewFailingStore(), db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{newsletter})
	nr.Setup(srv)

	data := url.Values{}
	data.Set(common.ParamNewsletter, newsletter)
	data.Set(common.ParamEmail, "bar@foo.com")

	req, err := http.NewRequest("POST", common.SubscribeEndpoint, strings.NewReader(data.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestConfirmSubscribeFailingStore(t *testing.T) {
	srv := http.NewServeMux()

	nr := NewTestNewsResource(NewFailingStore(), db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)

	data := url.Values{}
	data.Set(common.ParamNewsletter, testNewsletter)
	data.Set(common.ParamToken, common.Sign(secret, testEmail))

	req, err := http.NewRequest("GET", common.ConfirmEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func TestConfirmSubscribeFailingStore2(t *testing.T) {
	srv := http.NewServeMux()

	store := NewFailingStore()
	store.failGetSubscriber = false
	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)

	data := url.Values{}
	data.Set(common.ParamNewsletter, testNewsletter)
	data.Set(common.ParamToken, common.Sign(secret, testEmail))

	req, err := http.NewRequest("GET", common.ConfirmEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func TestConfirmSubscribeWithoutToken(t *testing.T) {
	srv := http.NewServeMux()

	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)

	data := url.Values{}
	data.Set(common.ParamNewsletter, testNewsletter)

	req, err := http.NewRequest("GET", common.ConfirmEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func TestConfirmSubscribeIncorrectNewsletter(t *testing.T) {
	srv := http.NewServeMux()

	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.ConfirmEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add(common.ParamNewsletter, "foo")
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func TestConfirmSubscribe(t *testing.T) {
	srv := http.NewServeMux()

	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)
	nr.ConfirmRedirectURL = testUrl

	req, err := http.NewRequest("GET", common.ConfirmEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	time.Sleep(10 * time.Nanosecond)
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusFound {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}

	l, err := resp.Location()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != testUrl {
		t.Errorf("Path does not match. expected=%v actual=%v", l.Path, testUrl)
	}

	i, _ := store.GetSubscriber(testNewsletter, testEmail)
	if !i.Confirmed() {
		t.Errorf("Confirm time not updated. created=%v confirm=%v", i.CreatedAt, i.ConfirmedAt)
	}
}

func TestGetSubscribersUnauthorized(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.SubscribersEndpoint, nil)
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
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.SubscribersEndpoint, nil)
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
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.SubscribersEndpoint, nil)
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
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.SubscribersEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add(common.ParamNewsletter, "test")
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

func TestGetSubscribersFailingStore(t *testing.T) {
	srv := http.NewServeMux()

	nr := NewTestAdminResource(NewFailingStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)
	nr.AddNewsletters([]string{testNewsletter})

	req, err := http.NewRequest("GET", common.SubscribersEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	req.URL.RawQuery = q.Encode()

	req.SetBasicAuth("any username", apiToken)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func TestGetSubscribersOK(t *testing.T) {
	srv := http.NewServeMux()

	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestAdminResource(store, db.NewNotificationsMapStore())
	nr.Setup(srv)
	nr.AddNewsletters([]string{testNewsletter})

	req, err := http.NewRequest("GET", common.SubscribersEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
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
	nr := NewTestNewsResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("POST", common.UnsubscribeEndpoint, nil)
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
	nr := NewTestNewsResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.UnsubscribeEndpoint, nil)
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

	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.UnsubscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	if store.Count() != 1 {
		t.Errorf("Wrong number of subscribers: %v", store.Count())
	}
}

func TestUnsubscribeWithBadToken(t *testing.T) {
	srv := http.NewServeMux()

	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.UnsubscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add(common.ParamNewsletter, "random value")
	q.Add(common.ParamToken, "abcde")
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	if store.Count() != 1 {
		t.Errorf("Wrong number of subscribers: %v", store.Count())
	}
}

func TestUnsubscribeFailingStore(t *testing.T) {
	srv := http.NewServeMux()

	nr := NewTestNewsResource(NewFailingStore(), db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.UnsubscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestUnsubscribe(t *testing.T) {
	srv := http.NewServeMux()

	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)
	nr.UnsubscribeRedirectURL = testUrl

	req, err := http.NewRequest("GET", common.UnsubscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	time.Sleep(10 * time.Nanosecond)
	srv.ServeHTTP(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	l, err := resp.Location()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != testUrl {
		t.Errorf("Path does not match. expected=%v actual=%v", l.Path, testUrl)
	}

	if store.Count() != 1 {
		t.Errorf("Wrong number of subscribers left: %d", store.Count())
	}

	i, _ := store.GetSubscriber(testNewsletter, testEmail)
	if !i.Unsubscribed() {
		t.Errorf("Unsubscribe time not updated. created=%v unsubscribe=%v", i.CreatedAt, i.UnsubscribedAt)
	}
}

func TestPostSubscribers(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("POST", common.SubscribersEndpoint, nil)
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

func TestPutSubscribersUnauthorized(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("PUT", common.SubscribersEndpoint, nil)
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

func TestPutSubscribersInvalidMedia(t *testing.T) {
	newsletter := "TestNewsletter"

	srv := http.NewServeMux()
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

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

	req, err := http.NewRequest("PUT", common.SubscribersEndpoint, bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("any username", apiToken)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func TestPutSubscribersWrongNewsletter(t *testing.T) {
	newsletter := "TestNewsletter"

	srv := http.NewServeMux()
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

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

	req, err := http.NewRequest("PUT", common.SubscribersEndpoint, bytes.NewBuffer(data))
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

func TestPutSubscribersFailingStore(t *testing.T) {
	newsletter := "TestNewsletter"

	srv := http.NewServeMux()
	nr := NewTestAdminResource(NewFailingStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)
	nr.AddNewsletters([]string{newsletter})

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

	req, err := http.NewRequest("PUT", common.SubscribersEndpoint, bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("any username", apiToken)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func PutSubscribersBaseSuite(subscribers []*common.Subscriber, store common.SubscribersStore) (*http.Response, error) {
	srv := http.NewServeMux()
	nr := NewTestAdminResource(store, db.NewNotificationsMapStore())
	nr.Setup(srv)
	nr.AddNewsletters([]string{testNewsletter})

	data, err := json.Marshal(subscribers)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", common.SubscribersEndpoint, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("any username", apiToken)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()
	return resp, nil
}

func PutSubscribersSuite(subscribers []*common.Subscriber, t *testing.T) {
	subscribersMap := make(map[string]*common.Subscriber)
	for _, s := range subscribers {
		subscribersMap[s.Email] = s
	}
	store := db.NewSubscribersMapStore()
	resp, err := PutSubscribersBaseSuite(subscribers, store)
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}

	for k, s := range subscribersMap {
		es, err := store.GetSubscriber(testNewsletter, k)
		if err != nil {
			t.Errorf("Email not imported: %v", k)
		}
		if es.Confirmed() != s.Confirmed() {
			t.Fatalf("Confirmed status does not match. email=%v", k)
		}
		if es.Unsubscribed() != s.Unsubscribed() {
			t.Fatalf("Unsubscribed status does not match. email=%v", k)
		}
	}
}

func TestPutUnconfirmedSubscribers(t *testing.T) {
	var subscribers []*common.Subscriber
	for i := 0; i < 10; i++ {
		s := &common.Subscriber{
			Newsletter:     testNewsletter,
			Email:          fmt.Sprintf("foo%v@bar.com", i),
			CreatedAt:      common.JsonTimeNow(),
			UnsubscribedAt: incorrectTime,
			ConfirmedAt:    incorrectTime,
		}
		subscribers = append(subscribers, s)
	}
	PutSubscribersSuite(subscribers, t)
}

func TestPutConfirmedSubscribers(t *testing.T) {
	var subscribers []*common.Subscriber
	for i := 0; i < 10; i++ {
		jt := common.JSONTime(time.Now().UTC().Add(-1 * time.Second))
		s := &common.Subscriber{
			Newsletter:     testNewsletter,
			Email:          fmt.Sprintf("foo%v@bar.com", i),
			CreatedAt:      jt,
			UnsubscribedAt: incorrectTime,
			ConfirmedAt:    incorrectTime,
		}
		s.ConfirmedAt = common.JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
		subscribers = append(subscribers, s)
	}
	PutSubscribersSuite(subscribers, t)
}

func TestPutUnsubscribedSubscribers(t *testing.T) {
	var subscribers []*common.Subscriber
	for i := 0; i < 10; i++ {
		jt := common.JSONTime(time.Now().UTC().Add(-1 * time.Second))
		s := &common.Subscriber{
			Newsletter:     testNewsletter,
			Email:          fmt.Sprintf("foo%v@bar.com", i),
			CreatedAt:      jt,
			UnsubscribedAt: incorrectTime,
			ConfirmedAt:    incorrectTime,
		}
		s.UnsubscribedAt = common.JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
		subscribers = append(subscribers, s)
	}
	PutSubscribersSuite(subscribers, t)
}

func TestGetComplaintsUnauthorized(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.ComplaintsEndpoint, nil)
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

func TestGetComplaintsFailingStore(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), &FailingNotificationsStore{})
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.ComplaintsEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	req.SetBasicAuth("any username", apiToken)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestGetComplaintsOK(t *testing.T) {
	srv := http.NewServeMux()
	store := db.NewNotificationsMapStore()
	store.AddBounce(testEmail, "from@email.com", false /*is transient*/)
	store.AddComplaint(testEmail, "from@email.com")
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), store)
	nr.Setup(srv)

	req, err := http.NewRequest("GET", common.ComplaintsEndpoint, nil)
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

func TestDeleteSubscribersUnauthorized(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	req, err := http.NewRequest("DELETE", common.SubscribersEndpoint, nil)
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

func TestDeleteSubscribersInvalidMedia(t *testing.T) {
	srv := http.NewServeMux()
	nr := NewTestAdminResource(db.NewSubscribersMapStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	keys := []*common.SubscriberKey{
		&common.SubscriberKey{
			Newsletter: testNewsletter,
			Email:      "email1@email.com",
		},
	}

	data, err := json.Marshal(keys)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("DELETE", common.SubscribersEndpoint, bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}

	req.SetBasicAuth("any username", apiToken)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("Unexpected status code: %d", resp.StatusCode)
	}
}

func TestDeleteSubscribers(t *testing.T) {
	srv := http.NewServeMux()
	store := db.NewSubscribersMapStore()
	for i := 0; i < 10; i++ {
		store.AddSubscriber(testNewsletter, fmt.Sprintf("email%v@email.com", i), testName)
	}

	nr := NewTestAdminResource(store, db.NewNotificationsMapStore())
	nr.Setup(srv)

	keys := []*common.SubscriberKey{
		&common.SubscriberKey{
			Newsletter: testNewsletter,
			Email:      "email1@email.com",
		},
		&common.SubscriberKey{
			Newsletter: testNewsletter,
			Email:      "email3@email.com",
		},
	}

	data, err := json.Marshal(keys)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("DELETE", common.SubscribersEndpoint, bytes.NewBuffer(data))
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

	if store.Count() != 8 {
		t.Errorf("Items were not deleted")
	}
}

func TestDeleteSubscribersFailingStore(t *testing.T) {
	srv := http.NewServeMux()

	nr := NewTestAdminResource(NewFailingStore(), db.NewNotificationsMapStore())
	nr.Setup(srv)

	keys := []*common.SubscriberKey{
		&common.SubscriberKey{
			Newsletter: testNewsletter,
			Email:      "email1@email.com",
		},
		&common.SubscriberKey{
			Newsletter: testNewsletter,
			Email:      "email3@email.com",
		},
	}

	data, err := json.Marshal(keys)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("DELETE", common.SubscribersEndpoint, bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth("any username", apiToken)

	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}
}

func TestSubscribeAlreadyConfirmed(t *testing.T) {
	srv := http.NewServeMux()
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)
	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)
	nr.ConfirmRedirectURL = testUrl

	s, _ := store.GetSubscriber(testNewsletter, testEmail)
	s.ConfirmedAt = common.JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
	if !s.Confirmed() {
		t.Errorf("Confirmed() is not updated")
	}

	data := url.Values{}
	data.Set(common.ParamNewsletter, testNewsletter)
	data.Set(common.ParamEmail, testEmail)

	req, err := http.NewRequest("POST", common.SubscribeEndpoint, strings.NewReader(data.Encode()))
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

	l, err := resp.Location()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != testUrl {
		t.Errorf("Path does not match. expected=%v actual=%v", l.Path, testUrl)
	}
}

func TestSubscribeAlreadyUnsubscribed(t *testing.T) {
	srv := http.NewServeMux()
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)
	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)
	nr.SubscribeRedirectURL = testUrl

	s, _ := store.GetSubscriber(testNewsletter, testEmail)
	s.UnsubscribedAt = common.JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
	if !s.Unsubscribed() {
		t.Errorf("Unsubscribed() is not updated")
	}

	data := url.Values{}
	data.Set(common.ParamNewsletter, testNewsletter)
	data.Set(common.ParamEmail, testEmail)

	req, err := http.NewRequest("POST", common.SubscribeEndpoint, strings.NewReader(data.Encode()))
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

	l, err := resp.Location()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != testUrl {
		t.Errorf("Path does not match. expected=%v actual=%v", l.Path, testUrl)
	}
}

func TestSubscribeAlreadyUnsubscribedAndConfirmed(t *testing.T) {
	srv := http.NewServeMux()
	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)
	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)
	nr.SubscribeRedirectURL = testUrl

	s, _ := store.GetSubscriber(testNewsletter, testEmail)
	s.UnsubscribedAt = common.JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
	s.ConfirmedAt = common.JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
	if !s.Unsubscribed() && !s.Confirmed() {
		t.Errorf("Unsubscribed() and Confirmed() are not updated")
	}

	data := url.Values{}
	data.Set(common.ParamNewsletter, testNewsletter)
	data.Set(common.ParamEmail, testEmail)

	req, err := http.NewRequest("POST", common.SubscribeEndpoint, strings.NewReader(data.Encode()))
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

	l, err := resp.Location()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != testUrl {
		t.Errorf("Path does not match. expected=%v actual=%v", l.Path, testUrl)
	}
}

func TestUnsubscribeNotSubscribedYet(t *testing.T) {
	srv := http.NewServeMux()

	store := db.NewSubscribersMapStore()

	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)
	nr.UnsubscribeRedirectURL = testUrl

	req, err := http.NewRequest("GET", common.UnsubscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	time.Sleep(10 * time.Nanosecond)
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}
}

func TestUnsubscribeUnsubscribed(t *testing.T) {
	srv := http.NewServeMux()

	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)

	s, _ := store.GetSubscriber(testNewsletter, testEmail)
	s.UnsubscribedAt = common.JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
	if !s.Unsubscribed() {
		t.Errorf("Unsubscribed() is not updated")
	}

	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)
	nr.UnsubscribeRedirectURL = testUrl

	req, err := http.NewRequest("GET", common.UnsubscribeEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	time.Sleep(10 * time.Nanosecond)
	srv.ServeHTTP(w, req)
	resp := w.Result()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	l, err := resp.Location()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != testUrl {
		t.Errorf("Path does not match. expected=%v actual=%v", l.Path, testUrl)
	}
}

func TestConfirmUnsubscribed(t *testing.T) {
	srv := http.NewServeMux()

	store := db.NewSubscribersMapStore()
	store.AddSubscriber(testNewsletter, testEmail, testName)
	s, _ := store.GetSubscriber(testNewsletter, testEmail)
	s.UnsubscribedAt = common.JSONTime(s.CreatedAt.Time().Add(1 * time.Second))
	if !s.Unsubscribed() {
		t.Errorf("Unsubscribed() is not updated")
	}

	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)
	nr.UnsubscribeRedirectURL = testUrl

	req, err := http.NewRequest("GET", common.ConfirmEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	time.Sleep(10 * time.Nanosecond)
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusFound {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}

	l, err := resp.Location()
	if err != nil {
		t.Fatal(err)
	}

	if l.String() != testUrl {
		t.Errorf("Path does not match. expected=%v actual=%v", l.Path, testUrl)
	}

	i, _ := store.GetSubscriber(testNewsletter, testEmail)
	if i.Confirmed() {
		t.Errorf("Confirm time was updated. created=%v confirm=%v", i.CreatedAt, i.ConfirmedAt)
	}
}

func TestConfirmMissing(t *testing.T) {
	srv := http.NewServeMux()
	store := db.NewSubscribersMapStore()
	nr := NewTestNewsResource(store, db.NewNotificationsMapStore())
	nr.AddNewsletters([]string{testNewsletter})
	nr.Setup(srv)
	nr.UnsubscribeRedirectURL = testUrl

	req, err := http.NewRequest("GET", common.ConfirmEndpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	q := req.URL.Query()
	q.Add(common.ParamNewsletter, testNewsletter)
	q.Add(common.ParamToken, common.Sign(secret, testEmail))
	req.URL.RawQuery = q.Encode()

	w := httptest.NewRecorder()
	time.Sleep(10 * time.Nanosecond)
	srv.ServeHTTP(w, req)

	resp := w.Result()

	if resp.StatusCode != http.StatusInternalServerError {
		body, _ := ioutil.ReadAll(resp.Body)
		t.Errorf("Unexpected status code: %d, body: %v", resp.StatusCode, string(body))
	}

	if store.Count() != 0 {
		t.Errorf("Wrong count in store: %v", store.Count())
	}
}
