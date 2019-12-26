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

*Listing* is a small and simple service that allows to self-host email subscriptions list on AWS using Lambda, DynamoDB, SES and SNS. It handles email list subscribe/unsubscribe/confirm actions as well as email bounces and complaints (if you use same SES account for sending the newsletter).

### What problem it solves?

Self-hosted email marketing solutions one can find today like [tinycampaign](https://github.com/parkerj/tinycampaign), [audience](https://github.com/aniftyco/audience), [mail-for-good](https://github.com/freeCodeCamp/mail-for-good), [colossus](https://github.com/vitorfs/colossus) and others have a few problems:

*   a need to run a webserver all of the time (resources are expensive)
*   operational overhead (certificate, updates, security)
*   they solve too many problems at the same time (subscription list, email sending, analytics, A/B testing, email templates)

There's also a [MoonMail](https://github.com/MoonMail/MoonMail) which is kind of a good approximation of a better system, but it is too complex to deploy and it tries to solve all problems at the same time as well. And there is almost no documentation since they are interested in selling SaaS version of it.

*Listing* is different. **It focuses only on building subscription list.** It is simple, well documented and easy to install.

As for other functionality, you can achieve analytics and A/B testing using systems like [Google Analytics](https://google.com/analytics) or others. Finding [good](https://github.com/InterNations/antwort) [email](https://github.com/leemunroe/responsive-html-email-template) [templates](https://github.com/mailgun/transactional-email-templates) or building ones using [available](http://mosaico.io/) [tools](https://beefree.io/) is also not a problem. And there are [many](https://github.com/rykov/paperboy) [ways](https://github.com/Circle-gg/thunder-mail) you can send those emails without a need to waste cloud computer resources at all other times.

### How does it work

*Listing* manages arbitrary amount of subscription lists using few lambda functions and DynamoDB table.

*Listing* uses so-called "double-confirmation" system where user has to confirm it's email even after entering it on your website and pressing "Subscribe" button. The reason is that this is way more reliable in terms of loyal audience building.

Subscriptions are stored in a DynamoDB table and are managed by 4 lambda functions with endpoints `/subscribe`, `/unsubscribe`, `/confirm` and `/subscribers`. The last endpoint is protected with BasicAuth for "admin" access (export/import of subscribers).

Bounces and Complaints are stored in additional DynamoDB table. This table is managed by another lambda function that is subscribed to SNS topic. AWS SES for a specific domain is a publisher to this SNS topic.

## How to use Listing

1. Deploy *Listing* (described below)
2. Test from the command line that everything works as expected (described below)
3. Create html form for your website to subscribe to emails using *Listing* Lambda URL

Everything else (confirmation, unsubscribe) will be handled in the emails or automatically.

The only thing left to do is to send periodic emails to your subscription list.

## How to avoid vendor lock

*Listing* is heavily dependent on AWS, but the size of DynamoDB table even with 100k subscribers is so small, you can safely export it to json using `/subscribers` endpoint and restore to any other cloud.

*Listing* uses serverless framework so migration to a different cloud will be less painful than from barebones AWS SAM templates.

## Deployment

Except of quite obvious prerequisites, the typical deployment procedure is as easy as editing `secrets.json` and running `serverless deploy` in the root of the repository. See detailed steps below.

### Generic prerequisites

*   Buy/Own a custom domain
*   [Create an AWS account](https://aws.amazon.com/premiumsupport/knowledge-center/create-and-activate-aws-account/)
*   Configure and [verify your domain in AWS SES](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/verify-domain-procedure.html) (Simple Email Service)
*   [Exit "sandbox" mode in SES](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/request-production-access.html) (you will have to contact support)
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

Most of the properties are self-descriptive. Redirect URLs are urls where user will be redirected to after pressing "Confirm", "Subscribe" or "Unsubscribe" buttons. `confirmUrl` is an url of one of the lambda functions used for email confirmation (can be arbitrary since it's edited after deployment). `emailFrom` is an email that will be used to send this confirmation email. `supportedNewsletters` is semicolon-separated list of newsletter names. *Listing* will ignore all subscribe/unsubscribe requests for newsletters that are not in this list.

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

In order to exit "sandbox" mode in AWS SES you need to have a procedure for handling bounces and complaints. *Listing* provides this functionality, but you have to do 1 manual action.

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
