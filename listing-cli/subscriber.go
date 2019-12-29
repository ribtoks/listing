package main

type SubscriberEx struct {
	Name         string `json:"name"`
	Newsletter   string `json:"newsletter"`
	Email        string `json:"email"`
	Confirmed    bool   `json:"confirmed"`
	Unsubscribed bool   `json:"unsubscribed"`
	Token        string `json:"token"`
}
