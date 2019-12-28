package db

import (
	"errors"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/ribtoks/backoff"
	"github.com/ribtoks/listing/pkg/common"
)

var (
	incorrectTime  = common.JSONTime(time.Unix(1, 1))
	errChunkTooBig = errors.New("Chunk of data contains more than allowed 25 items")
)

const (
	dynamoDBChunkSize = 25
)

// NewSubscribersStore creates an instance of SubscribersDynamoDB struct
func NewSubscribersStore(table string, sess *session.Session) *SubscribersDynamoDB {
	return &SubscribersDynamoDB{
		Client:    dynamodb.New(sess),
		TableName: table,
	}
}

type SubscribersDynamoDB struct {
	TableName string
	Client    dynamodbiface.DynamoDBAPI
}

// make sure SubscribersDynamoDB implements interface
var _ common.SubscribersStore = (*SubscribersDynamoDB)(nil)

func (s *SubscribersDynamoDB) AddSubscriber(newsletter, email, name string) error {
	i, err := dynamodbattribute.MarshalMap(common.Subscriber{
		Name:           name,
		Newsletter:     newsletter,
		Email:          email,
		CreatedAt:      common.JsonTimeNow(),
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

func (s *SubscribersDynamoDB) RemoveSubscriber(newsletter, email string) error {
	updateVal := struct {
		UnsubscribedAt common.JSONTime `json:":unsubscribed_at"`
	}{
		UnsubscribedAt: common.JsonTimeNow(),
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

func (s *SubscribersDynamoDB) Subscribers(newsletter string) (subscribers []*common.Subscriber, err error) {
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
		var items []*common.Subscriber
		err := dynamodbattribute.UnmarshalListOfMaps(page.Items, &items)
		if err != nil {
			// print the error and continue receiving pages
			log.Printf("Could not unmarshal AWS data. err=%v", err)
			return true
		}

		subscribers = append(subscribers, items...)
		// continue receiving pages (can be used to limit the number of pages)
		return true
	})

	return
}

func (s *SubscribersDynamoDB) AddSubscribersChunk(subscribers []*common.Subscriber) error {
	// AWS DynamoDB restriction
	if len(subscribers) > dynamoDBChunkSize {
		return errChunkTooBig
	}

	requests := make([]*dynamodb.WriteRequest, 0, len(subscribers))
	for _, i := range subscribers {
		attr, err := dynamodbattribute.MarshalMap(i)
		if err != nil {
			return err
		}

		requests = append(requests, &dynamodb.WriteRequest{
			PutRequest: &dynamodb.PutRequest{
				Item: attr,
			},
		})
	}

	b := &backoff.Backoff{
		Min:    100 * time.Millisecond,
		Max:    1 * time.Second,
		Factor: 2,
		Jitter: false,
	}

	for len(requests) > 0 {
		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				s.TableName: requests,
			},
		}
		res, err := s.Client.BatchWriteItem(input)
		if err != nil {
			return err
		}
		if unprocessed, ok := res.UnprocessedItems[s.TableName]; ok {
			log.Printf("Found unprocessed items. count=%v", len(unprocessed))
			requests = unprocessed
		} else {
			break
		}
		time.Sleep(b.Duration())
	}
	return nil
}

func (s *SubscribersDynamoDB) AddSubscribers(subscribers []*common.Subscriber) error {
	for i := 0; i < len(subscribers); i += dynamoDBChunkSize {
		end := i + dynamoDBChunkSize

		if end > len(subscribers) {
			end = len(subscribers)
		}

		err := s.AddSubscribersChunk(subscribers[i:end])
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SubscribersDynamoDB) ConfirmSubscriber(newsletter, email string) error {
	updateVal := struct {
		ConfirmedAt common.JSONTime `json:":confirmed_at"`
	}{
		ConfirmedAt: common.JsonTimeNow(),
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

func (s *SubscribersDynamoDB) DeleteSubscribersChunk(keys []*common.SubscriberKey) error {
	// AWS DynamoDB restriction
	if len(keys) > dynamoDBChunkSize {
		return errChunkTooBig
	}

	requests := make([]*dynamodb.WriteRequest, 0, len(keys))
	for _, k := range keys {
		attr, err := dynamodbattribute.MarshalMap(k)
		if err != nil {
			return err
		}

		requests = append(requests, &dynamodb.WriteRequest{
			DeleteRequest: &dynamodb.DeleteRequest{
				Key: attr,
			},
		})
	}

	b := &backoff.Backoff{
		Min:    100 * time.Millisecond,
		Max:    1 * time.Second,
		Factor: 2,
		Jitter: false,
	}

	for len(requests) > 0 {
		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]*dynamodb.WriteRequest{
				s.TableName: requests,
			},
		}
		res, err := s.Client.BatchWriteItem(input)
		if err != nil {
			return err
		}
		if unprocessed, ok := res.UnprocessedItems[s.TableName]; ok {
			log.Printf("Found unprocessed items. count=%v", len(unprocessed))
			requests = unprocessed
		} else {
			break
		}
		time.Sleep(b.Duration())
	}

	return nil
}

func (s *SubscribersDynamoDB) DeleteSubscribers(keys []*common.SubscriberKey) error {
	for i := 0; i < len(keys); i += dynamoDBChunkSize {
		end := i + dynamoDBChunkSize

		if end > len(keys) {
			end = len(keys)
		}

		err := s.DeleteSubscribersChunk(keys[i:end])
		if err != nil {
			return err
		}
	}
	return nil
}