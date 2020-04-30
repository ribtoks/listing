package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
	"github.com/ribtoks/listing/pkg/api"
	"github.com/ribtoks/listing/pkg/db"
)

var (
	handlerLambda *httpadapter.HandlerAdapter
)

// Handler is the main entry point to this lambda
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return handlerLambda.ProxyWithContext(ctx, req)
}

func main() {
	apiToken := os.Getenv("API_TOKEN")
	subscribersTableName := os.Getenv("SUBSCRIBERS_TABLE")
	notificationsTableName := os.Getenv("NOTIFICATIONS_TABLE")
	supportedNewsletters := os.Getenv("SUPPORTED_NEWSLETTERS")

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION")),
	})

	if err != nil {
		log.Fatalf("Failed to create AWS session. err=%v", err)
	}

	subscribers := db.NewSubscribersStore(subscribersTableName, sess)
	notifications := db.NewNotificationsStore(notificationsTableName, sess)

	router := http.NewServeMux()
	newsletter := &api.AdminResource{
		APIToken:      apiToken,
		Subscribers:   subscribers,
		Notifications: notifications,
		Newsletters:   make(map[string]bool),
	}

	sn := strings.Split(supportedNewsletters, ";")
	newsletter.AddNewsletters(sn)

	newsletter.Setup(router)
	handlerLambda = httpadapter.New(router)

	lambda.Start(Handler)
}
