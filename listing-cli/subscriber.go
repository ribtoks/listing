package main

import "github.com/ribtoks/listing/pkg/common"

type SubscriberEx struct {
	Name         string `json:"name"`
	Newsletter   string `json:"newsletter"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	Confirmed    bool   `json:"confirmed"`
	Unsubscribed bool   `json:"unsubscribed"`
}

func NewSubscriberEx(s *common.Subscriber, secret string) *SubscriberEx {
	return &SubscriberEx{
		Name:         s.Name,
		Newsletter:   s.Newsletter,
		Email:        s.Email,
		Confirmed:    s.Confirmed(),
		Unsubscribed: s.Unsubscribed(),
		Token:        common.Sign(secret, s.Email),
	}
}
