package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ribtoks/checkmail"
)

// NewsletterResource manages http requests and data storage
// for newsletter subscriptions
type NewsletterResource struct {
	apiToken               string
	secret                 string
	subscribeRedirectUrl   string
	unsubscribeRedirectUrl string
	confirmRedirectUrl     string
	confirmUrl             string
	newsletters            map[string]bool
	store                  Store
	mailer                 Mailer
}

const (
	paramNewsletter = "newsletter"
	paramToken      = "token"
	// assume there cannot be such a huge http requests for subscription
	kilobyte             = 1024
	megabyte             = 1024 * kilobyte
	maxSubscribeBodySize = kilobyte / 2
	maxImportBodySize    = 25 * megabyte
)

func (nr *NewsletterResource) setup(router *http.ServeMux) {
	router.HandleFunc("/subscribers", nr.auth(nr.subscribers))
	router.HandleFunc("/subscribe", nr.method("POST", nr.subscribe))
	router.HandleFunc("/unsubscribe", nr.method("GET", nr.unsubscribe))
	router.HandleFunc("/confirm", nr.method("GET", nr.confirm))
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

	emails, err := nr.store.GetSubscribers(newsletter)
	if err != nil {
		log.Printf("error fetching subscribers: %v\n", err)
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

	var subscribers []*Subscriber
	err := dec.Decode(&subscribers)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	ss := make([]*Subscriber, 0, len(subscribers))
	for _, s := range subscribers {
		if nr.isValidNewsletter(s.Newsletter) {
			ss = append(ss, s)
		}
	}

	if len(ss) == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = nr.store.AddSubscribers(ss)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (nr *NewsletterResource) subscribers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		{
			nr.getSubscribers(w, r)
		}
	case "PUT":
		{
			nr.putSubscribers(w, r)
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
		log.Printf("error parsing form: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	newsletter := r.FormValue(paramNewsletter)
	email := r.FormValue("email")

	err = checkmail.ValidateFormat(email)
	if err != nil {
		log.Printf("error validating email: %q", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if !nr.isValidNewsletter(newsletter) {
		log.Printf("Invalid newsletter: %v", newsletter)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = nr.store.AddSubscriber(newsletter, email)
	if err != nil {
		log.Printf("error subscribing email %q to %q: %v", email, newsletter, err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	log.Printf("subscribed email %q to %q", email, newsletter)

	nr.mailer.SendConfirmation(newsletter, email, nr.confirmUrl)

	w.Header().Set("Location", nr.subscribeRedirectUrl)
	http.Redirect(w, r, nr.subscribeRedirectUrl, http.StatusFound)
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

	email, ok := Unsign(nr.secret, unsubscribeToken)
	if !ok {
		log.Printf("error unsigning %q", unsubscribeToken)
		http.Error(w, "Invalid unsubscribe token", http.StatusBadRequest)
		return
	}

	err := nr.store.RemoveSubscriber(newsletter, email)
	if err != nil {
		log.Printf("error unsubscribing %q: %v", email, err)
		http.Error(w, "Error unsubscribing from newsletter", http.StatusInternalServerError)
		return
	}

	log.Printf("unsubscribed email %q from %q", email, newsletter)
	w.Header().Set("Location", nr.unsubscribeRedirectUrl)
	http.Redirect(w, r, nr.unsubscribeRedirectUrl, http.StatusFound)
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

	email, ok := Unsign(nr.secret, subscribeToken)
	if !ok {
		log.Printf("error unsigning %q", subscribeToken)
		http.Error(w, "Invalid subscribe token", http.StatusBadRequest)
		return
	}

	err := nr.store.ConfirmSubscriber(newsletter, email)
	if err != nil {
		log.Printf("error confirming %q: %v", email, err)
		http.Error(w, "Error confirming subscription", http.StatusInternalServerError)
		return
	}

	log.Printf("confirmed email %q from %q", email, newsletter)
	w.Header().Set("Location", nr.confirmRedirectUrl)
	http.Redirect(w, r, nr.confirmRedirectUrl, http.StatusFound)
}
