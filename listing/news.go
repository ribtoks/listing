// Package news provides a very simple DynamoDB-backed mailing list for newsletters.
package main

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

// item model.
type Subscriber struct {
	Newsletter     string   `json:"newsletter"`
	Email          string   `json:"email"`
	CreatedAt      JSONTime `json:"created_at"`
	UnsubscribedAt JSONTime `json:"unsubscribed_at"`
	ConfirmedAt    JSONTime `json:"confirmed_at"`
	ComplainedAt   JSONTime `json:"complained_at"`
	BouncedAt      JSONTime `json:"bounced_at"`
}

func NewStore(table string, sess *session.Session) *DynamoDBStore {
	return &DynamoDBStore{
		Client:    dynamodb.New(sess),
		TableName: table,
	}
}

type Store interface {
	AddSubscriber(newsletter, email string) error
	RemoveSubscriber(newsletter, email string) error
	GetSubscribers(newsletter string) (subscribers []*Subscriber, err error)
	ConfirmSubscriber(newsletter, email string) error
}

// DynamoDBStore is a DynamoDB mailing list storage implementation.
type DynamoDBStore struct {
	TableName string
	Client    dynamodbiface.DynamoDBAPI
}

// make sure DynamoDBStore implements interface
var _ Store = (*DynamoDBStore)(nil)

// AddSubscriber adds a subscriber to a newsletter.
func (s *DynamoDBStore) AddSubscriber(newsletter, email string) error {
	i, err := dynamodbattribute.MarshalMap(Subscriber{
		Newsletter:     newsletter,
		Email:          email,
		CreatedAt:      JSONTime(time.Now()),
		ConfirmedAt:    JSONTime(time.Unix(1, 1)),
		ComplainedAt:   JSONTime(time.Unix(1, 1)),
		UnsubscribedAt: JSONTime(time.Unix(1, 1)),
		BouncedAt:      JSONTime(time.Unix(1, 1)),
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

	return nil
}

func (s *DynamoDBStore) RemoveSubscriber(newsletter, email string) error {
	unsubscribeTime := JSONTime(time.Now()).String()
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":unsubscribeTime": {
				S: aws.String(unsubscribeTime),
			},
		},
		TableName: &s.TableName,
		Key: map[string]*dynamodb.AttributeValue{
			"newsletter": &dynamodb.AttributeValue{
				S: &newsletter,
			},
			"email": &dynamodb.AttributeValue{
				S: &email,
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set unsubscribed_at = :unsubscribeTime"),
	}
	_, err := s.Client.UpdateItem(input)
	return err
}

// GetSubscribers returns subscriber emails for a newsletter.
func (s *DynamoDBStore) GetSubscribers(newsletter string) (subscribers []*Subscriber, err error) {
	query := &dynamodb.QueryInput{
		TableName:              &s.TableName,
		KeyConditionExpression: aws.String(`newsletter = :newsletter`),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":newsletter": &dynamodb.AttributeValue{
				S: &newsletter,
			},
		},
	}

	err = s.Client.QueryPages(query, func(page *dynamodb.QueryOutput, more bool) bool {
		var items []*Subscriber
		err := dynamodbattribute.UnmarshalListOfMaps(page.Items, &items)
		if err != nil {
			// print the error and continue receiving pages
			log.Printf("\nCould not unmarshal AWS data: err = %v\n", err)
			return true
		}

		subscribers = append(subscribers, items...)
		// continue receiving pages (can be used to limit the number of pages)
		return true
	})

	return
}

func (s *DynamoDBStore) ConfirmSubscriber(newsletter, email string) error {
	confirmTime := JSONTime(time.Now()).String()
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":confirmTime": {
				S: aws.String(confirmTime),
			},
		},
		TableName: &s.TableName,
		Key: map[string]*dynamodb.AttributeValue{
			"newsletter": &dynamodb.AttributeValue{
				S: &newsletter,
			},
			"email": &dynamodb.AttributeValue{
				S: &email,
			},
		},
		ReturnValues:     aws.String("UPDATED_NEW"),
		UpdateExpression: aws.String("set confirmed_at = :confirmTime"),
	}
	_, err := s.Client.UpdateItem(input)
	return err
}
