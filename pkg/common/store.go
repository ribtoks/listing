package common

// SubscribersStore is an interface used to manage subscribers DB from the main API
type SubscribersStore interface {
	AddSubscriber(newsletter, email, name string) error
	RemoveSubscriber(newsletter, email string) error
	Subscribers(newsletter string) (subscribers []*Subscriber, err error)
	AddSubscribers(subscribers []*Subscriber) error
	DeleteSubscribers(keys []*SubscriberKey) error
	ConfirmSubscriber(newsletter, email string) error
	GetSubscriber(newsletter, email string) (*Subscriber, error)
}

// Mailer is an interface for sending confirmation emails for subscriptions
type Mailer interface {
	SendConfirmation(newsletter, email, name, confirmURL string) error
}

// NotificationsStore is an interface used to manage SES bounce and complaint
// notifications from sesnotify API
type NotificationsStore interface {
	AddBounce(email, from string, isTransient bool) error
	AddComplaint(email, from string) error
	Notifications() (notifications []*SesNotification, err error)
}
