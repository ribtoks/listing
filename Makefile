.PHONY: build clean deploy

build:
	dep ensure -v
	env GOOS=linux go build -ldflags="-s -w" -o bin/listing listing/main.go listing/store.go listing/token.go listing/api.go listing/email.go listing/confirm_html.go listing/jsontime.go listing/subscriber.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/sesnotify sesnotify/main.go sesnotify/sesmessage.go sesnotify/store.go

clean:
	rm -rf ./bin ./vendor ./.serverless Gopkg.lock

deploy: clean build
	sls deploy --config serverless-api.yml --stage dev --region "us-east-1" --verbose

local: clean build
	./scripts/deploy_local.sh
