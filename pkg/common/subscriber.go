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

func (s *Subscriber) Confirmed() bool {
	return s.ConfirmedAt.Time().Sub(s.CreatedAt.Time()) > 0
}

func (s *Subscriber) Unsubscribed() bool {
	return s.UnsubscribedAt.Time().Sub(s.CreatedAt.Time()) > 0
}

// SubscriberKey is used for deletion of subscribers
type SubscriberKey struct {
	Newsletter string `json:"newsletter"`
	Email      string `json:"email"`
}
