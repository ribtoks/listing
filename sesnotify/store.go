package main

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

const (
	softBounceType = "soft_bounce"
	hardBounceType = "hard_bounce"
	complaintType  = "complaint"
)

// JSONTime is an alias that allows standartized
// serialization and deserialization with strings
type JSONTime time.Time

func (t *JSONTime) MarshalJSON() ([]byte, error) {
	ct := time.Time(*t)
	str := fmt.Sprintf("%q", ct.Format(time.RFC3339))
	return []byte(str), nil
}

type sesNotification struct {
	Email        string   `json:"email"`
	From         string   `json:"from"`
	ReceivedAt   JSONTime `json:"received_at"`
	Notification string   `json:"notification"`
}

type Store interface {
	AddBounce(email, from string, isTransient bool) error
	AddComplaint(email, from string) error
}

// DynamoDBStore is an implementation of Store interface
// that is capable of working with AWS DynamoDB
type DynamoDBStore struct {
	TableName string
	Client    dynamodbiface.DynamoDBAPI
}

func NewStore(table string, sess *session.Session) *DynamoDBStore {
	return &DynamoDBStore{
		Client:    dynamodb.New(sess),
		TableName: table,
	}
}

// make sure DynamoDBStore implements interface
var _ Store = (*DynamoDBStore)(nil)

func (s *DynamoDBStore) StoreNotification(email, from string, t string) error {
	i, err := dynamodbattribute.MarshalMap(sesNotification{
		Email:        email,
		ReceivedAt:   JSONTime(time.Now().UTC()),
		Notification: t,
		From:         from,
	})

	if err != nil {
		return err
	}

	_, err = s.Client.PutItem(&dynamodb.PutItemInput{
		TableName: &s.TableName,
		Item:      i,
	})

	if err != nil {
		return err
	}

	log.Printf("Stored notification email=%v type=%v", email, t)
	return nil
}

func (s *DynamoDBStore) AddBounce(email, from string, isTransient bool) error {
	bounceType := softBounceType
	if !isTransient {
		bounceType = hardBounceType
	}
	return s.StoreNotification(email, from, bounceType)
}

func (s *DynamoDBStore) AddComplaint(email, from string) error {
	return s.StoreNotification(email, from, complaintType)
}
