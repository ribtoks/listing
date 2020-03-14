package common

import "github.com/rs/xid"

// Subscriber incapsulates newsletter subscriber information
// stored in the DynamoDB table
type Subscriber struct {
	Name           string   `json:"name,omitempty"`
	Newsletter     string   `json:"newsletter"`
	Email          string   `json:"email"`
	CreatedAt      JSONTime `json:"created_at"`
	UnsubscribedAt JSONTime `json:"unsubscribed_at"`
	ConfirmedAt    JSONTime `json:"confirmed_at"`
	UserID         string   `json:"user_id,omitempty"`
}

// Confirmed checks if subscriber has confirmed the email via link
func (s *Subscriber) Confirmed() bool {
	return s.ConfirmedAt.Time().After(s.CreatedAt.Time())
}

// Unsubscribed checks if subscriber pressed "Unsubscribe" link
func (s *Subscriber) Unsubscribed() bool {
	return s.UnsubscribedAt.Time().After(s.CreatedAt.Time())
}

func (s *Subscriber) Validate() {
	if len(s.UserID) > 0 {
		return
	}

	guid := xid.New()
	s.UserID = guid.String()
}

// SubscriberKey is used for deletion of subscribers
type SubscriberKey struct {
	Newsletter string `json:"newsletter"`
	Email      string `json:"email"`
}

type SubscriberEx struct {
	Name         string `json:"name" yaml:"name"`
	Newsletter   string `json:"newsletter" yaml:"newsletter"`
	Email        string `json:"email" yaml:"email"`
	Token        string `json:"token" yaml:"token"`
	Confirmed    bool   `json:"confirmed" yaml:"confirmed"`
	Unsubscribed bool   `json:"unsubscribed" yaml:"unsubscribed"`
	UserID       string `json:"user_id" yaml:"user_id"`
}

func NewSubscriberEx(s *Subscriber, secret string) *SubscriberEx {
	return &SubscriberEx{
		Name:         s.Name,
		Newsletter:   s.Newsletter,
		Email:        s.Email,
		Confirmed:    s.Confirmed(),
		Unsubscribed: s.Unsubscribed(),
		Token:        Sign(secret, s.Email),
		UserID:       s.UserID,
	}
}
