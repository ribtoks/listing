package common

// SubscribersStore is an interface used to manage subscribers DB from the main API
type SubscribersStore interface {
	AddSubscriber(newsletter, email, name string) error
	RemoveSubscriber(newsletter, email string) error
	GetSubscribers(newsletter string) (subscribers []*Subscriber, err error)
	AddSubscribers(subscribers []*Subscriber) error
	ConfirmSubscriber(newsletter, email string) error
}

// Mailer is an interface for sending confirmation emails for subscriptions
type Mailer interface {
	SendConfirmation(newsletter, email, name, confirmURL string) error
}

// NotificationsStore is an interface used to manage SES bounce and complaint
// notifications from sesnotify API
type NotificationStore interface {
	AddBounce(email, from string, isTransient bool) error
	AddComplaint(email, from string) error
}
