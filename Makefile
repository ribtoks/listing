.PHONY: build clean deploy

STAGE ?= dev
REGION ?= eu-west-1

test:
	go test ./...

build:
	dep ensure -v
	go build -o cmd/listing-cli/listing-cli cmd/listing-cli/*.go
	go build -o cmd/listing-send/listing-send cmd/listing-send/*.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/listing cmd/listing/*.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/sesnotify cmd/sesnotify/*.go
	env GOOS=linux go build -ldflags="-s -w" -o bin/ladmin cmd/ladmin/*.go

clean:
	rm -rf ./bin ./vendor ./.serverless Gopkg.lock

domain:
	sls create_domain --stage '${STAGE}' --region '$(REGION)' --config serverless-api.yml

deploy-db:
	sls deploy --config serverless-db.yml --stage '$(STAGE)' --region '$(REGION)' --verbose

remove-db:
	sls remove --config serverless-db.yml --stage '$(STAGE)' --region '$(REGION)' --verbose

deploy-api:
	sls deploy --config serverless-api.yml --stage '$(STAGE)' --region '$(REGION)' --verbose

remove-api:
	sls remove --config serverless-api.yml --stage '$(STAGE)' --region '$(REGION)' --verbose

deploy-admin:
	sls deploy --config serverless-admin.yml --stage '$(STAGE)' --region '$(REGION)' --verbose

remove-admin:
	sls remove --config serverless-admin.yml --stage '$(STAGE)' --region '$(REGION)' --verbose

deploy-all: clean build deploy-db deploy-api deploy-admin
	echo "Done for stage=${STAGE} region=${REGION}"

remove-all: remove-api remove-admin remove-db
	echo "Done for stage=${STAGE} region=${REGION}"
