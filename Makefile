.PHONY: build clean deploy

build:
	dep ensure -v
	env GOOS=linux go build -ldflags="-s -w" -o bin/listing listing/main.go listing/news.go listing/token.go listing/api.go

clean:
	rm -rf ./bin ./vendor ./.serverless Gopkg.lock

deploy: clean build
	sls deploy --stage dev --region "us-east-1" --verbose

local: clean build
	./scripts/deploy_local.sh
