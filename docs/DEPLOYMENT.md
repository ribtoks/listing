Except of quite obvious prerequisites, the typical deployment procedure is as easy as editing `secrets.json` and running `serverless deploy` in the root of the repository. See detailed steps below.

## Generic prerequisites

*   Buy/Own a custom domain
*   [Create an AWS account](https://aws.amazon.com/premiumsupport/knowledge-center/create-and-activate-aws-account/)
*   Configure and [verify your domain in AWS SES](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-domain-procedure.html) (Simple Email Service)
*   [Exit "sandbox" mode in SES](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/request-production-access.html) (you will have to contact support)
*   Create "confirm"/"unsubscribed" pages on your website

## Install prerequisites

*   [Go](https://golang.org/dl/)
*   [Node.js](https://nodejs.org/en/download/) (for npm and serverless)
*   [Serverless](https://serverless.com/framework/docs/getting-started/) (`npm install -g serverless`)
*   Go dep (run `go get -u github.com/golang/dep/cmd/dep` anywhere)
*   Serverless plugins (run `npm install` in repository root)
*   [Configure AWS credentials](https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/setup-credentials.html)

## Edit secrets

Copy example file to `secrets.json`

`cp secrets.json.example secrets.json`

and edit it.

Most of the properties are self-descriptive. Redirect URLs are urls where user will be redirected to after pressing "Confirm", "Subscribe" or "Unsubscribe" buttons. `confirmUrl` is an url of one of the lambda functions used for email confirmation (can be arbitrary since it's edited after deployment). `emailFrom` is an email that will be used to send this confirmation email. `supportedNewsletters` is semicolon-separated list of newsletter names. *Listing* will ignore all subscribe/unsubscribe requests for newsletters that are not in this list.

## Deploy DB and SNS topic

Config file `serverless-db.yml` describes resources that will not be frequently changed. Usually you would deploy them only once.

Use `make deploy-db` or

`serverless deploy --config serverless-db.yml`

This command will deploy resources to `dev` stage and to `eu-west-1` region so if you want to use something else, you have to add parameters `--stage dev --region "eu-west-1"`.

## Deploy API

Config file `serverless-api.yml` contains definitions of lambda functions written in Go. If you are a developer, you may want to redeploy them frequently during development.

You can use `make deploy-api` to run this step or you can use 2 commands:

```
make build
serverless deploy --config serverless-api.yml
```

This command will deploy resources to `dev` stage and to `eu-west-1` region so if you want to use something else, you have to add parameters `--stage dev --region "eu-west-1"`.

## Deploy Admin API

Config file `serverless-admin.yml` contains definitions of lambda functions with Admin APIs like getting or setting subscribers. This functionality might be redeployed even more frequently than ordinary API

You can use `make deploy-admin` to run this step or you can use 2 commands:

```
make build
serverless deploy --config serverless-admin.yml
```

This command will deploy resources to `dev` stage and to `eu-west-1` region so if you want to use something else, you have to add parameters `--stage dev --region "eu-west-1"`.

## Configure confirm and redirect URLs

Go to AWS Console UI and in Lambda section find `listing-subscribe` function. Set `CONFIRM_URL` in it's environmental variables to point to the API Gateway address for `listing-confirm` function's address.

(optional - you can do that in the end) Configure `SUBSCRIBE_REDIRECT_URL`, `UNSUBSCRIBE_REDIRECT_URL`, `CONFIRM_REDIRECT_URL` in the appropriate lambda function to point to the pages on your website.

## Configure SNS topic for bounces and complaints

In order to exit "sandbox" mode in AWS SES you need to have a procedure for handling bounces and complaints. *Listing* provides this functionality, but you have to do 1 manual action.

Go to [AWS Console UI and set Bounce and Complaint](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/configure-sns-notifications.html) SNS topic's ARN for your SES domain to the `listing-ses-notifications` topic. You can find it in `SES -> Domains -> (select your domain) -> Notifications`. Arn will be an output of `serverless deploy` command for `serverless-db.yml` config. Example of such ARN: `arn:aws:sns:eu-west-1:1234567890:dev-listing-ses-notifications`.

## Configure redirect URLs to your website

Open lambda function properties in the AWS management console and set all `_REDIRECT_URL` variables to the appropriate values that point to your website. You will need to have pages for email confirmation, unsubscribe confirmation etc.

