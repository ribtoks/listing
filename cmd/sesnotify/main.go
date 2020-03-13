package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/ribtoks/listing/pkg/common"
	"github.com/ribtoks/listing/pkg/db"
)

var (
	store common.NotificationsStore
)

func handler(ctx context.Context, snsEvent events.SNSEvent) {
	log.Printf("Processing %v records", len(snsEvent.Records))

	for _, record := range snsEvent.Records {
		snsRecord := record.SNS
		var sesMessage common.SesMessage
		err := json.Unmarshal([]byte(snsRecord.Message), &sesMessage)
		if err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}

		switch sesMessage.NotificationType {
		case "Bounce":
			{
				isTransient := sesMessage.Bounce.BounceType == "Transient"
				for _, r := range sesMessage.Bounce.BouncedRecipients {
					err = store.AddBounce(r.EmailAddress, sesMessage.Mail.Source, isTransient)
					if err != nil {
						log.Printf("Failed to add bounce: %v", err)
					}
				}
			}
		case "Complaint":
			{
				for _, r := range sesMessage.Bounce.BouncedRecipients {
					err = store.AddComplaint(r.EmailAddress, sesMessage.Mail.Source)
					if err != nil {
						log.Printf("Failed to add complaint: %v", err)
					}
				}
			}
		default:
			{
				log.Printf("Unexpected message type: %v", sesMessage.NotificationType)
			}
		}
	}
}

func main() {
	tableName := os.Getenv("NOTIFICATIONS_TABLE")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})

	if err != nil {
		log.Fatalf("Failed to create AWS session. err=%v", err)
	}

	store = db.NewNotificationsStore(tableName, sess)

	lambda.Start(handler)
}
