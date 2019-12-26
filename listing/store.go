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

var (
	incorrectTime = JSONTime(time.Unix(1, 1))
)

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
	AddSubscribers(subscribers []*Subscriber) error
	ConfirmSubscriber(newsletter, email string) error
}

type DynamoDBStore struct {
	TableName string
	Client    dynamodbiface.DynamoDBAPI
}

// make sure DynamoDBStore implements interface
var _ Store = (*DynamoDBStore)(nil)

func (s *DynamoDBStore) AddSubscriber(newsletter, email string) error {
	i, err := dynamodbattribute.MarshalMap(Subscriber{
		Newsletter:     newsletter,
		Email:          email,
		CreatedAt:      jsonTimeNow(),
		UnsubscribedAt: incorrectTime,
		ConfirmedAt:    incorrectTime,
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
	updateVal := struct {
		UnsubscribedAt JSONTime `json:":unsubscribed_at"`
	}{
		UnsubscribedAt: jsonTimeNow(),
	}

	update, err := dynamodbattribute.MarshalMap(updateVal)
	if err != nil {
		return err
	}
	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: update,
		UpdateExpression:          aws.String("set unsubscribed_at = :unsubscribed_at"),
		TableName:                 &s.TableName,
		Key: map[string]*dynamodb.AttributeValue{
			"newsletter": &dynamodb.AttributeValue{
				S: &newsletter,
			},
			"email": &dynamodb.AttributeValue{
				S: &email,
			},
		},
		ReturnValues: aws.String("UPDATED_NEW"),
	}
	_, err = s.Client.UpdateItem(input)
	return err
}

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

func (s *DynamoDBStore) AddSubscribers(subscribers []*Subscriber) error {
	return nil
}

func (s *DynamoDBStore) ConfirmSubscriber(newsletter, email string) error {
	updateVal := struct {
		ConfirmedAt JSONTime `json:":confirmed_at"`
	}{
		ConfirmedAt: jsonTimeNow(),
	}

	update, err := dynamodbattribute.MarshalMap(updateVal)
	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		ExpressionAttributeValues: update,
		UpdateExpression:          aws.String("set confirmed_at = :confirmed_at"),
		TableName:                 &s.TableName,
		Key: map[string]*dynamodb.AttributeValue{
			"newsletter": &dynamodb.AttributeValue{
				S: &newsletter,
			},
			"email": &dynamodb.AttributeValue{
				S: &email,
			},
		},
		ReturnValues: aws.String("UPDATED_NEW"),
	}
	_, err = s.Client.UpdateItem(input)
	return err
}
