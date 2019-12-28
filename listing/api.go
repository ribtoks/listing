package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/ribtoks/checkmail"
	"github.com/ribtoks/listing/pkg/common"
)

// NewsletterResource manages http requests and data storage
// for newsletter subscriptions
type NewsletterResource struct {
	apiToken               string
	secret                 string
	subscribeRedirectURL   string
	unsubscribeRedirectURL string
	confirmRedirectURL     string
	confirmURL             string
	newsletters            map[string]bool
	subscribers            common.SubscribersStore
	notifications          common.NotificationsStore
	mailer                 common.Mailer
}

const (
	paramNewsletter = "newsletter"
	paramToken      = "token"
	// assume there cannot be such a huge http requests for subscription
	kilobyte             = 1024
	megabyte             = 1024 * kilobyte
	maxSubscribeBodySize = kilobyte / 2
	maxImportBodySize    = 25 * megabyte
	maxDeleteBodySize    = 5 * megabyte
)

func (nr *NewsletterResource) setup(router *http.ServeMux) {
	router.HandleFunc(common.SubscribersEndpoint, nr.auth(nr.serveSubscribers))
	router.HandleFunc(common.ComplaintsEndpoint, nr.auth(nr.complaints))
	router.HandleFunc(common.SubscribeEndpoint, nr.method("POST", nr.subscribe))
	router.HandleFunc(common.UnsubscribeEndpoint, nr.method("GET", nr.unsubscribe))
	router.HandleFunc(common.ConfirmEndpoint, nr.method("GET", nr.confirm))
}

func (nr *NewsletterResource) addNewsletters(n []string) {
	for _, i := range n {
		nr.newsletters[i] = true
	}
}

// auth middleware.
func (nr *NewsletterResource) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, pass, ok := r.BasicAuth()
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		if pass != nr.apiToken {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}
}

func (nr *NewsletterResource) method(m string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != m {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		next.ServeHTTP(w, r)
	}
}

func (nr *NewsletterResource) isValidNewsletter(n string) bool {
	if n == "" {
		return false
	}
	_, ok := nr.newsletters[n]
	return ok
}

func (nr *NewsletterResource) getSubscribers(w http.ResponseWriter, r *http.Request) {
	newsletter := r.URL.Query().Get(paramNewsletter)

	if !nr.isValidNewsletter(newsletter) {
		http.Error(w, "The newsletter parameter is invalid", http.StatusBadRequest)
		return
	}

	emails, err := nr.subscribers.Subscribers(newsletter)
	if err != nil {
		log.Printf("Failed to fetch subscribers. err=%v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(emails)
}

func (nr *NewsletterResource) putSubscribers(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type header is not application/json", http.StatusUnsupportedMediaType)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxImportBodySize)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var subscribers []*common.Subscriber
	err := dec.Decode(&subscribers)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	ss := make([]*common.Subscriber, 0, len(subscribers))
	for _, s := range subscribers {
		if !nr.isValidNewsletter(s.Newsletter) {
			log.Printf("Skipping unsupported newsletter. value=%v", s.Newsletter)
			continue
		}
		if err = checkmail.ValidateFormat(s.Email); err != nil {
			log.Printf("Skipping invalid email. value=%v", s.Email)
			continue
		}
		s.CreatedAt = common.JsonTimeNow()
		ss = append(ss, s)
	}

	if len(ss) == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = nr.subscribers.AddSubscribers(ss)
	if err != nil {
		log.Printf("Failed to import subscribers. err=%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (nr *NewsletterResource) deleteSubscribers(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Content-Type header is not application/json", http.StatusUnsupportedMediaType)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxDeleteBodySize)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var keys []*common.SubscriberKey
	err := dec.Decode(&keys)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if len(keys) == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = nr.subscribers.DeleteSubscribers(keys)
	if err != nil {
		log.Printf("Failed to delete subscribers. err=%v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (nr *NewsletterResource) serveSubscribers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		{
			nr.getSubscribers(w, r)
		}
	case "PUT":
		{
			nr.putSubscribers(w, r)
		}
	case "DELETE":
		{
			nr.deleteSubscribers(w, r)
		}
	default:
		{
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
}

func (nr *NewsletterResource) subscribe(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSubscribeBodySize)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Failed to parse form. err=%v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	newsletter := r.FormValue(paramNewsletter)
	email := r.FormValue("email")

	err = checkmail.ValidateFormat(email)
	if err != nil {
		log.Printf("Failed to validate email. err=%q", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !nr.isValidNewsletter(newsletter) {
		log.Printf("Invalid newsletter. value=%v", newsletter)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// name is optional
	name := strings.TrimSpace(r.FormValue("name"))

	err = nr.subscribers.AddSubscriber(newsletter, email, name)
	if err != nil {
		log.Printf("Failed to add subscription. email=%q newsletter=%q name=%v err=%v", email, newsletter, name, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	log.Printf("Added subscription email=%q newsletter=%q name=%v", email, newsletter, name)

	nr.mailer.SendConfirmation(newsletter, email, name, nr.confirmURL)

	w.Header().Set("Location", nr.subscribeRedirectURL)
	http.Redirect(w, r, nr.subscribeRedirectURL, http.StatusFound)
}

// unsubscribe route.
func (nr *NewsletterResource) unsubscribe(w http.ResponseWriter, r *http.Request) {
	newsletter := r.URL.Query().Get(paramNewsletter)
	unsubscribeToken := r.URL.Query().Get(paramToken)

	if newsletter == "" {
		http.Error(w, "The newsletter query-string parameter is required", http.StatusBadRequest)
		return
	}

	if !nr.isValidNewsletter(newsletter) {
		http.Error(w, "Invalid newsletter param", http.StatusBadRequest)
		return
	}

	if unsubscribeToken == "" {
		http.Error(w, "The token query-string parameter is required", http.StatusBadRequest)
		return
	}

	email, ok := common.Unsign(nr.secret, unsubscribeToken)
	if !ok {
		log.Printf("Failed to unsign token. value=%q", unsubscribeToken)
		http.Error(w, "Invalid unsubscribe token", http.StatusBadRequest)
		return
	}

	err := nr.subscribers.RemoveSubscriber(newsletter, email)
	if err != nil {
		log.Printf("Failed to unsubscribe. email=%q err=%v", email, err)
		http.Error(w, "Error unsubscribing from newsletter", http.StatusInternalServerError)
		return
	}

	log.Printf("Unsubscribed. email=%q newsletter=%q", email, newsletter)
	w.Header().Set("Location", nr.unsubscribeRedirectURL)
	http.Redirect(w, r, nr.unsubscribeRedirectURL, http.StatusFound)
}

func (nr *NewsletterResource) confirm(w http.ResponseWriter, r *http.Request) {
	newsletter := r.URL.Query().Get(paramNewsletter)
	subscribeToken := r.URL.Query().Get(paramToken)

	if !nr.isValidNewsletter(newsletter) {
		http.Error(w, "Invalid newsletter param", http.StatusBadRequest)
		return
	}

	if subscribeToken == "" {
		http.Error(w, "The token query-string parameter is required", http.StatusBadRequest)
		return
	}

	email, ok := common.Unsign(nr.secret, subscribeToken)
	if !ok {
		log.Printf("Failed to unsign token. value=%q", subscribeToken)
		http.Error(w, "Invalid subscribe token", http.StatusBadRequest)
		return
	}

	err := nr.subscribers.ConfirmSubscriber(newsletter, email)
	if err != nil {
		log.Printf("Failed to confirm subscription. email=%q err=%v", email, err)
		http.Error(w, "Error confirming subscription", http.StatusInternalServerError)
		return
	}

	log.Printf("Confirmed subscription. email=%q newsletter=%q", email, newsletter)
	w.Header().Set("Location", nr.confirmRedirectURL)
	http.Redirect(w, r, nr.confirmRedirectURL, http.StatusFound)
}

func (nr *NewsletterResource) complaints(w http.ResponseWriter, r *http.Request) {
	notifications, err := nr.notifications.Notifications()
	if err != nil {
		log.Printf("Failed to fetch notifications. err=%v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(notifications)

}
