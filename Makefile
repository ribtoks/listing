.PHONY: build clean deploy

test:
	go test ./...

build:
	dep ensure -v
	go build -o cmd/listing-cli/listing-cli cmd/listing-cli/*.go
	go build -o cmd/listing-send/listing-send cmd/listing-send/*.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/listing cmd/listing/*.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/sesnotify cmd/sesnotify/*.go

clean:
	rm -rf ./bin ./vendor ./.serverless Gopkg.lock

deploy: clean build
	sls deploy --config serverless-api.yml --stage dev --region "us-east-1" --verbose

local: clean build
	./scripts/deploy_local.sh
