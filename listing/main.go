package main

import (
	"context"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
)

var (
	handlerLambda *httpadapter.HandlerAdapter
	newsletter    *NewsletterResource
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse
type Request events.APIGatewayProxyRequest

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return handlerLambda.ProxyWithContext(ctx, req)
}

func main() {
	secret := os.Getenv("TOKEN_SECRET")
	apiToken := os.Getenv("API_TOKEN")
	subscribeURL := os.Getenv("SUBSCRIBE_REDIRECT_URL")
	unsubscribeURL := os.Getenv("UNSUBSCRIBE_REDIRECT_URL")
	tableName := os.Getenv("DYNAMO_TABLE")

	router := http.NewServeMux()
	newsletter := &NewsletterResource{
		apiToken:       apiToken,
		secret:         secret,
		subscribeURL:   subscribeURL,
		unsubscribeURL: unsubscribeURL,
		store:          NewStore(tableName),
	}

	newsletter.Setup(router)
	handlerLambda = httpadapter.New(router)

	lambda.Start(Handler)
}
