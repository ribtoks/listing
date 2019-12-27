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
