// Package news provides a very simple DynamoDB-backed mailing list for newsletters.
package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
)

// item model.
type Subscriber struct {
	Newsletter     string    `json:"newsletter"`
	Email          string    `json:"email"`
	CreatedAt      time.Time `json:"created_at"`
	UnsubscribedAt time.Time `json:"unsubscribed_at"`
	ConfirmedAt    time.Time `json:"confirmed_at"`
	ComplainedAt   time.Time `json:"complained_at"`
	BouncedAt      time.Time `json:"bounced_at"`
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
		Newsletter:   newsletter,
		Email:        email,
		CreatedAt:    time.Now(),
		ConfirmedAt:  time.Unix(1, 1),
		ComplainedAt: time.Unix(1, 1),
		BouncedAt:    time.Unix(1, 1),
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
	unsubscribeTime := time.Now().String()
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
		for _, item := range page.Items {
			i := &Subscriber{}
			if err := dynamodbattribute.UnmarshalMap(item, &i); err == nil {
				subscribers = append(subscribers, i)
			}
		}
		return true
	})

	return
}

func (s *DynamoDBStore) ConfirmSubscriber(newsletter, email string) error {
	confirmTime := time.Now().String()
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
