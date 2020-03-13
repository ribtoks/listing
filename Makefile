.PHONY: build clean deploy

test:
	go test ./...

build:
	dep ensure -v
	go build -o cmd/listing-cli/listing-cli cmd/listing-cli/main.go cmd/listing-cli/client.go cmd/listing-cli/export.go cmd/listing-cli/printer.go cmd/listing-cli/subscribe.go cmd/listing-cli/complaints.go cmd/listing-cli/unsubscribe.go cmd/listing-cli/import.go cmd/listing-cli/delete.go
	go build -o cmd/listing-send/listing-send cmd/listing-send/main.go cmd/listing-send/campaign.go cmd/listing-send/sender.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/listing cmd/listing/main.go cmd/listing/email.go cmd/listing/confirm_html.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/sesnotify cmd/sesnotify/main.go

clean:
	rm -rf ./bin ./vendor ./.serverless Gopkg.lock

deploy: clean build
	sls deploy --config serverless-api.yml --stage dev --region "us-east-1" --verbose

local: clean build
	./scripts/deploy_local.sh
