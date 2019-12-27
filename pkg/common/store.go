package common

type SubscribersStore interface {
	AddSubscriber(newsletter, email string) error
	RemoveSubscriber(newsletter, email string) error
	GetSubscribers(newsletter string) (subscribers []*Subscriber, err error)
	AddSubscribers(subscribers []*Subscriber) error
	ConfirmSubscriber(newsletter, email string) error
}

// Mailer is an interface to sending confirmation emails for subscriptions
type Mailer interface {
	SendConfirmation(newsletter, email string, confirmURL string) error
}

type NotificationStore interface {
	AddBounce(email, from string, isTransient bool) error
	AddComplaint(email, from string) error
}
