package db

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbiface"
	"github.com/ribtoks/listing/pkg/common"
)

const (
	SoftBounceType = "soft_bounce"
	HardBounceType = "hard_bounce"
	ComplaintType  = "complaint"
)

// NotificationsDynamoDB is an implementation of Store interface
// that is capable of working with AWS DynamoDB
type NotificationsDynamoDB struct {
	TableName string
	Client    dynamodbiface.DynamoDBAPI
}

var _ common.NotificationsStore = (*NotificationsDynamoDB)(nil)

// NewNotificationsStore returns new instance of NotificationsDynamoDB
func NewNotificationsStore(table string, sess *session.Session) *NotificationsDynamoDB {
	return &NotificationsDynamoDB{
		Client:    dynamodb.New(sess),
		TableName: table,
	}
}

func (s *NotificationsDynamoDB) StoreNotification(email, from string, t string) error {
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

func (s *NotificationsDynamoDB) AddBounce(email, from string, isTransient bool) error {
	bounceType := SoftBounceType
	if !isTransient {
		bounceType = HardBounceType
	}
	return s.StoreNotification(email, from, bounceType)
}

func (s *NotificationsDynamoDB) AddComplaint(email, from string) error {
	return s.StoreNotification(email, from, ComplaintType)
}

func (s *NotificationsDynamoDB) Notifications() (notifications []*common.SesNotification, err error) {
	query := &dynamodb.QueryInput{
		TableName: &s.TableName,
	}

	err = s.Client.QueryPages(query, func(page *dynamodb.QueryOutput, more bool) bool {
		var items []*common.SesNotification
		err := dynamodbattribute.UnmarshalListOfMaps(page.Items, &items)
		if err != nil {
			// print the error and continue receiving pages
			log.Printf("Could not unmarshal AWS data. err=%v", err)
			return true
		}

		notifications = append(notifications, items...)
		// continue receiving pages (can be used to limit the number of pages)
		return true
	})

	return
}
