package common

// Subscriber incapsulates newsletter subscriber information
// stored in the DynamoDB table
type Subscriber struct {
	Name           string   `json:"name"`
	Newsletter     string   `json:"newsletter"`
	Email          string   `json:"email"`
	CreatedAt      JSONTime `json:"created_at"`
	UnsubscribedAt JSONTime `json:"unsubscribed_at"`
	ConfirmedAt    JSONTime `json:"confirmed_at"`
}

// Confirmed checks if subscriber has confirmed the email via link
func (s *Subscriber) Confirmed() bool {
	return s.ConfirmedAt.Time().After(s.CreatedAt.Time())
}

// Unsubscribed checks if subscriber pressed "Unsubscribe" link
func (s *Subscriber) Unsubscribed() bool {
	return s.UnsubscribedAt.Time().After(s.CreatedAt.Time())
}

// SubscriberKey is used for deletion of subscribers
type SubscriberKey struct {
	Newsletter string `json:"newsletter"`
	Email      string `json:"email"`
}
