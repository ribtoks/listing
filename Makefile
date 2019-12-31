.PHONY: build clean deploy

test:
	go test ./...

build:
	dep ensure -v
	go build -o listing-cli/listing-cli listing-cli/main.go listing-cli/client.go listing-cli/export.go listing-cli/printer.go listing-cli/subscribe.go listing-cli/subscriber.go listing-cli/complaints.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/listing listing/main.go listing/email.go listing/confirm_html.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/sesnotify sesnotify/main.go

clean:
	rm -rf ./bin ./vendor ./.serverless Gopkg.lock

deploy: clean build
	sls deploy --config serverless-api.yml --stage dev --region "us-east-1" --verbose

local: clean build
	./scripts/deploy_local.sh
