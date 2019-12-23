package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/ribtoks/checkmail"
)

type NewsletterResource struct {
	apiToken       string
	secret         string
	subscribeURL   string
	unsubscribeURL string
	store          Store
}

func (nr *NewsletterResource) Setup(router *http.ServeMux) {
	router.HandleFunc("/subscribers", nr.auth(nr.subscribers))
	router.HandleFunc("/subscribe", nr.subscribe)
	router.HandleFunc("/unsubscribe", nr.unsubscribe)
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

// subscribers route.
func (nr *NewsletterResource) subscribers(w http.ResponseWriter, r *http.Request) {
	newsletter := r.URL.Query().Get("newsletter")

	if newsletter == "" {
		http.Error(w, "The newsletter query-string parameter is required", http.StatusBadRequest)
		return
	}

	emails, err := nr.store.GetSubscribers(newsletter)
	if err != nil {
		log.Printf("error fetching subscribers: %v\n", err)
		return
	}

	b, err := json.Marshal(emails)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

// subscribe route.
func (nr *NewsletterResource) subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err := r.ParseForm()
	if err != nil {
		log.Printf("error parsing form: %v", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	newsletter := r.FormValue("newsletter")
	email := r.FormValue("email")

	err = checkmail.ValidateFormat(email)
	if err != nil {
		log.Printf("error validating email: %q", err)
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
	w.Header().Set("Location", nr.subscribeURL)
	http.Redirect(w, r, nr.subscribeURL, http.StatusFound)
}

// unsubscribe route.
func (nr *NewsletterResource) unsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	newsletter := r.URL.Query().Get("newsletter")
	unsubscribeToken := r.URL.Query().Get("token")

	if newsletter == "" {
		http.Error(w, "The newsletter query-string parameter is required", http.StatusBadRequest)
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
	w.Header().Set("Location", nr.unsubscribeURL)
	http.Redirect(w, r, nr.unsubscribeURL, http.StatusFound)
}
