.PHONY: build clean deploy

test:
	go test -v -short ./...

build:
	dep ensure -v
	env GOOS=linux go build -ldflags="-s -w" -o bin/listing listing/main.go listing/api.go listing/email.go listing/confirm_html.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/sesnotify sesnotify/main.go

clean:
	rm -rf ./bin ./vendor ./.serverless Gopkg.lock

deploy: clean build
	sls deploy --config serverless-api.yml --stage dev --region "us-east-1" --verbose

local: clean build
	./scripts/deploy_local.sh
