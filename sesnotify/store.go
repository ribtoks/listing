package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/ribtoks/listing/pkg/common"
)

const (
	softBounceType = "soft_bounce"
	hardBounceType = "hard_bounce"
	complaintType  = "complaint"
)

// DynamoDBStore is an implementation of Store interface
// that is capable of working with AWS DynamoDB
type DynamoDBStore struct {
	TableName string
	Client    dynamodbiface.DynamoDBAPI
}

var _ common.NotificationStore = (*DynamoDBStore)(nil)

// NewStore returns new instance of DynamoDBStore
func NewStore(table string, sess *session.Session) *DynamoDBStore {
	return &DynamoDBStore{
		Client:    dynamodb.New(sess),
		TableName: table,
	}
}

func (s *DynamoDBStore) StoreNotification(email, from string, t string) error {
	i, err := dynamodbattribute.MarshalMap(common.SesNotification{
		Email:        email,
		ReceivedAt:   common.JsonTimeNow(),
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
