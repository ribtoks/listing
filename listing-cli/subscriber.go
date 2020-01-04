package main

import "github.com/ribtoks/listing/pkg/common"

type SubscriberEx struct {
	Name         string `json:"name" yaml:"name"`
	Newsletter   string `json:"newsletter" yaml:"newsletter"`
	Email        string `json:"email" yaml:"email"`
	Token        string `json:"token" yaml:"toke"`
	Confirmed    bool   `json:"confirmed" yaml:"confirmed"`
	Unsubscribed bool   `json:"unsubscribed" yaml:"unsubscribed"`
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
