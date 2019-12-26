# listing

[![Build Status](https://travis-ci.org/ribtoks/listing.svg?branch=master)](https://travis-ci.org/ribtoks/listing)
[![Build status](https://ci.appveyor.com/api/projects/status/ypmg5foasuiuf5lh/branch/master?svg=true)](https://ci.appveyor.com/project/Ribtoks/listing/branch/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/ribtoks/listing)](https://goreportcard.com/report/github.com/ribtoks/listing)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/7ca0882c24314f01afe10bb857449ccb)](https://www.codacy.com/manual/ribtoks/listing?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=ribtoks/listing&amp;utm_campaign=Badge_Grade)
[![Maintainability](https://api.codeclimate.com/v1/badges/084d620cf3ef2f84ce99/maintainability)](https://codeclimate.com/github/ribtoks/listing/maintainability)

![license](https://img.shields.io/badge/license-MIT-blue.svg)
![copyright](https://img.shields.io/badge/%C2%A9-Taras_Kushnir-blue.svg)
![language](https://img.shields.io/badge/language-go-blue.svg)

## About

*Listing* is a small and simple service that allows to self-host email subscriptions list on AWS using Lambda, DynamoDB, SES and SNS. It handles email list subscribe/unsubscribe actions as well as email bounces and complaints.

### What problem it solves?

All self-hosted email marketing solutions as of today like [tinycampaign](https://github.com/parkerj/tinycampaign), [audience](https://github.com/aniftyco/audience), [mail-for-good](https://github.com/freeCodeCamp/mail-for-good), [colossus](https://github.com/vitorfs/colossus) and others have few problems:

*   a need to run a webserver all of the time (resources are not free)
*   operational overhead (certificate, updates, security)
*   they solve too many problems at the same time (subscription list, email sending, analytics, A/B testing, email templates)

There's also [MoonMail](https://github.com/MoonMail/MoonMail) which is kind of a good approximation of a better system, but it is too complex to deploy and it tries to solve all problems at the same time as well. Also there is almost no documentation since they are interested in selling SaaS version of it.

*Listing* is different. It focuses only on subscription list. You can achieve analytics and A/B testing using systems like [Google Analytics](https://google.com/analytics) or others. Finding good email templates is also not a problem. And there are many ways you can send those emails without a need to waste cloud computer resources at all other times.

*Listing* uses AWS Lambda for managing subscribe/unsubscribe actions as well as bounces/complaints which are very well suited for this task since they are relatively rare events.

In order to send the emails to the subscription list managed by *Listing*, you will need a temporary resource (like your own computer or EC2 instance) and any software capable of sending emails via SMTP. One could recomment a [paperboy](https://github.com/rykov/paperboy) for that, but you can use even your favorite email client.

## Deployment

Except of quite obvious prerequisites, the typical deployment procedure is as easy as editing `secrets.json` and running `serverless deploy` in the root of the repository. See detailed steps below.

### Generic prerequisites

*   Buy/Own a custom domain
*   [Create an AWS account](https://aws.amazon.com/premiumsupport/knowledge-center/create-and-activate-aws-account/)
*   Configure and [verify your domain in AWS SES](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-domain-procedure.html) (Simple Email Service)
*   [Exit "sandbox" mode in SES](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/request-production-access.html) (you have to contact support)
*   Create "confirm"/"unsubscribed" pages on your website

### Install prerequisites:

*   [Go](https://golang.org/dl/)
*   [Node.js](https://nodejs.org/en/download/) (for npm and serverless)
*   [Serverless](https://serverless.com/framework/docs/getting-started/) (`npm install -g serverless`)
*   Serverless plugins (`npm install` in repository root)
*   [Configure AWS credentials](https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/setup-credentials.html)

### Edit secrets

Copy example file to `secrets.json`

`cp secrets.json.example secrets.json`

and edit it.

### Deploy DB and SNS topic

Config file `serverless-db.yml` describes resources that will not be frequently changed. Usually you would deploy them only once.

`serverless deploy --config serverless-db.yml`

In order to specify stage and region, add parameters `--stage dev --region "us-east-1"`.

### Deploy API

Config file `serverless-api.yml` contains definitions of lambda functions written in Go. If you are a developer, you may want to redeploy them frequently.

You can use `make deploy` to run this step or you can use 2 commands:

```
make build
serverless deploy --config serverless-api.yml
```

In order to specify stage and region, add parameters `--stage dev --region "us-east-1"`.

### Configure confirm and redirect URLs

Go to AWS Console UI and in Lambda section find `listing-subscribe` function. Set `CONFIRM_URL` in it's environmental variables to point to the API Gateway address for `listing-confirm` function's address.

(optional - you can do that in the end) Configure `SUBSCRIBE_REDIRECT_URL`, `UNSUBSCRIBE_REDIRECT_URL`, `CONFIRM_REDIRECT_URL` in the appropriate lambda function to point to the pages on your website.

### Configure SNS topic for bounces and complaints

Go to [AWS Console UI and set Bounce and Complaint](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/configure-sns-notifications.html) SNS topic's ARN for your SES domain to the `listing-ses-notifications` topic. You can find it in `SES -> Domains -> (select your domain) -> Notifications`. Arn will be an output of `serverless deploy` command for `serverless-db.yml` config. Example of such ARN: `arn:aws:sns:us-east-1:1234567890:dev-listing-ses-notifications`.

## Testing

### Subscribing to a newsletter

`curl -v --data "newsletter=NewsletterName&email=foo@bar.com" https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev/subscribe`

### Email confirmation

Run previous "subscription" command to the real email address and click "Confirm email" button. Go to DynamoDB tables in AWS Console UI and check `listing-subscribers` table.

### Listing all subscribers

`curl -v -u :your-api-token https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev/subscribers?newsletter=NewsletterName`

### Handling bounce and complaint

Send email to `bounce@simulator.amazonses.com` from SES UI in AWS Console and check DynamoDB table `listing-sesnotify` if it contains the bounce notification.

Send email to `complaint@simulator.amazonses.com` from SES UI in AWS Console and check DynamoDB table `listing-sesnotify` if it contains the complaint notification.
