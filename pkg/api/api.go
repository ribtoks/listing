package api

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
	ApiToken               string
	Secret                 string
	SubscribeRedirectURL   string
	UnsubscribeRedirectURL string
	ConfirmRedirectURL     string
	ConfirmURL             string
	Newsletters            map[string]bool
	Subscribers            common.SubscribersStore
	Notifications          common.NotificationsStore
	Mailer                 common.Mailer
}

const (
	// assume there cannot be such a huge http requests for subscription
	kilobyte             = 1024
	megabyte             = 1024 * kilobyte
	maxSubscribeBodySize = kilobyte / 2
	maxImportBodySize    = 25 * megabyte
	maxDeleteBodySize    = 5 * megabyte
)

func (nr *NewsletterResource) Setup(router *http.ServeMux) {
	router.HandleFunc(common.SubscribersEndpoint, nr.auth(nr.serveSubscribers))
	router.HandleFunc(common.ComplaintsEndpoint, nr.auth(nr.complaints))
	router.HandleFunc(common.SubscribeEndpoint, nr.method("POST", nr.subscribe))
	router.HandleFunc(common.UnsubscribeEndpoint, nr.method("GET", nr.unsubscribe))
	router.HandleFunc(common.ConfirmEndpoint, nr.method("GET", nr.confirm))
}

func (nr *NewsletterResource) AddNewsletters(n []string) {
	for _, i := range n {
		nr.Newsletters[i] = true
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

		if pass != nr.ApiToken {
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
	_, ok := nr.Newsletters[n]
	return ok
}

func (nr *NewsletterResource) getSubscribers(w http.ResponseWriter, r *http.Request) {
	newsletter := r.URL.Query().Get(common.ParamNewsletter)

	if !nr.isValidNewsletter(newsletter) {
		http.Error(w, "The newsletter parameter is invalid", http.StatusBadRequest)
		return
	}

	emails, err := nr.Subscribers.Subscribers(newsletter)
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

	err = nr.Subscribers.AddSubscribers(ss)
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
		log.Printf("Failed to decode keys. err=%v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = nr.Subscribers.DeleteSubscribers(keys)
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
			log.Printf("Unsupported method for subscribers. method=%v", r.Method)
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
	}
}

func (nr *NewsletterResource) validate(newsletter, email string) bool {
	err := checkmail.ValidateFormat(email)
	if err != nil {
		log.Printf("Failed to validate email. value=%q err=%q", email, err)
		return false
	}

	if !nr.isValidNewsletter(newsletter) {
		log.Printf("Invalid newsletter. value=%v", newsletter)
		return false
	}

	return true
}

func (nr *NewsletterResource) subscribe(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSubscribeBodySize)
	err := r.ParseForm()
	if err != nil {
		log.Printf("Failed to parse form. err=%v", err)
	}

	newsletter := r.FormValue(common.ParamNewsletter)
	email := r.FormValue(common.ParamEmail)

	if ok := nr.validate(newsletter, email); !ok {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if s, err := nr.Subscribers.GetSubscriber(newsletter, email); err == nil {
		log.Printf("Subscriber already exists. email=%v newsletter=%v", email, newsletter)
		if s.Confirmed() && !s.Unsubscribed() {
			log.Printf("Email is already confirmed. email=%v newsletter=%v confirmed_at=%v", email, newsletter, s.ConfirmedAt.Time())
			w.Header().Set("Location", nr.ConfirmRedirectURL)
			http.Redirect(w, r, nr.ConfirmRedirectURL, http.StatusFound)
			return
		}
	}

	// name is optional
	name := strings.TrimSpace(r.FormValue(common.ParamName))
	err = nr.Subscribers.AddSubscriber(newsletter, email, name)
	if err != nil {
		log.Printf("Failed to add subscription. email=%q newsletter=%q name=%v err=%v", email, newsletter, name, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	log.Printf("Added subscription email=%q newsletter=%q name=%v", email, newsletter, name)

	nr.Mailer.SendConfirmation(newsletter, email, name, nr.ConfirmURL)

	w.Header().Set("Location", nr.SubscribeRedirectURL)
	http.Redirect(w, r, nr.SubscribeRedirectURL, http.StatusFound)
}

// unsubscribe route.
func (nr *NewsletterResource) unsubscribe(w http.ResponseWriter, r *http.Request) {
	newsletter := r.URL.Query().Get(common.ParamNewsletter)
	unsubscribeToken := r.URL.Query().Get(common.ParamToken)

	if newsletter == "" {
		http.Error(w, "The newsletter query-string parameter is required", http.StatusBadRequest)
		return
	}

	if !nr.isValidNewsletter(newsletter) {
		http.Error(w, "Invalid newsletter param", http.StatusBadRequest)
		return
	}

	email, ok := common.Unsign(nr.Secret, unsubscribeToken)
	if !ok {
		log.Printf("Failed to unsign token. value=%q", unsubscribeToken)
		http.Error(w, "Invalid unsubscribe token", http.StatusBadRequest)
		return
	}

	err := nr.Subscribers.RemoveSubscriber(newsletter, email)
	if err != nil {
		log.Printf("Failed to unsubscribe. email=%q err=%v", email, err)
		http.Error(w, "Error unsubscribing from newsletter", http.StatusInternalServerError)
		return
	}

	log.Printf("Unsubscribed. email=%q newsletter=%q", email, newsletter)
	w.Header().Set("Location", nr.UnsubscribeRedirectURL)
	http.Redirect(w, r, nr.UnsubscribeRedirectURL, http.StatusFound)
}

func (nr *NewsletterResource) confirm(w http.ResponseWriter, r *http.Request) {
	newsletter := r.URL.Query().Get(common.ParamNewsletter)
	subscribeToken := r.URL.Query().Get(common.ParamToken)

	if !nr.isValidNewsletter(newsletter) {
		http.Error(w, "Invalid newsletter param", http.StatusBadRequest)
		return
	}

	email, ok := common.Unsign(nr.Secret, subscribeToken)
	if !ok {
		log.Printf("Failed to unsign token. value=%q", subscribeToken)
		http.Error(w, "Invalid subscribe token", http.StatusBadRequest)
		return
	}

	if s, err := nr.Subscribers.GetSubscriber(newsletter, email); err == nil {
		if s.Unsubscribed() {
			log.Printf("Subscriber has already unsubscribed. newsletter=%v email=%v", newsletter, email)
			w.Header().Set("Location", nr.UnsubscribeRedirectURL)
			http.Redirect(w, r, nr.UnsubscribeRedirectURL, http.StatusFound)
			return
		}
	} else {
		log.Printf("Subscriber cannot be found. newsletter=%v email=%v err=%v", newsletter, email, err)
		http.Error(w, "Error confirming subscription", http.StatusInternalServerError)
		return
	}

	err := nr.Subscribers.ConfirmSubscriber(newsletter, email)
	if err != nil {
		log.Printf("Failed to confirm subscription. email=%q err=%v", email, err)
		http.Error(w, "Error confirming subscription", http.StatusInternalServerError)
		return
	}

	log.Printf("Confirmed subscription. email=%q newsletter=%q", email, newsletter)
	w.Header().Set("Location", nr.ConfirmRedirectURL)
	http.Redirect(w, r, nr.ConfirmRedirectURL, http.StatusFound)
}

func (nr *NewsletterResource) complaints(w http.ResponseWriter, r *http.Request) {
	notifications, err := nr.Notifications.Notifications()
	if err != nil {
		log.Printf("Failed to fetch notifications. err=%v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(notifications)

}
