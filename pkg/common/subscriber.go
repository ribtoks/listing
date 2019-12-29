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
	d := s.ConfirmedAt.Time().Sub(s.CreatedAt.Time())
	return d.Nanoseconds() > 0
}

// Unsubscribed checks if subscriber pressed "Unsubscribe" link
func (s *Subscriber) Unsubscribed() bool {
	d := s.UnsubscribedAt.Time().Sub(s.CreatedAt.Time())
	return d.Nanoseconds() > 0
}

// SubscriberKey is used for deletion of subscribers
type SubscriberKey struct {
	Newsletter string `json:"newsletter"`
	Email      string `json:"email"`
}
