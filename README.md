# listing

## About

*Listing* is a very small and simple service that allows to self-host email subscriptions list on AWS using Lambda, DynamoDB, SES and SNS.

### What problem it solves?

All self-hosted email marketing solutions like [tinycampaign](https://github.com/parkerj/tinycampaign), [audience](https://github.com/aniftyco/audience), [mail-for-good](https://github.com/freeCodeCamp/mail-for-good), [colossus](https://github.com/vitorfs/colossus) have few problems:

*   a need to run a webserver all of the time (resources are not free)
*   operational overhead (certificate, updates, security)
*   they solve too many problems at the same time (subscriptions, sending, analytics, A/B testing)

There's also [MoonMail](https://github.com/MoonMail/MoonMail) which is kind of a good approximation of such system, but it is too complex to deploy (also there almost no documentation) and it also tries to solve all problems at the same time.

*Listing* is different. It focuses only on subscription list. You can achieve analytics and A/B testing using systems like [Google Analytics](https://google.com/analytics) or others. Also there are many ways you can send those emails and you don't need to waste computer resources at other times. *Listing* uses AWS Lambda for managing subscribe/unsubscribe actions as well as bounces/complaints which are very well suited for this task since they are relatively rare events.

## Deployment

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

`serverless deploy --config serverless-db.yml`

### Deploy API

`serverless deploy --config serverless-api.yml`

### Configure confirm and redirect URLs

Go to AWS Console UI and set `CONFIRM_URL` for `listing-subscribe` Lambda function to point to the API Gateway address for `listing-confirm` address.

(optional - you can do that in the end) Configure `SUBSCRIBE_REDIRECT_URL`, `UNSUBSCRIBE_REDIRECT_URL`, `CONFIRM_REDIRECT_URL` in the appropriate lambda function to point to the pages on your website.

### Configure SNS topic for bounces and complaints

Go to [AWS Console UI and set Bounce and Complaint](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/configure-sns-notifications.html) SNS topic's ARN for your SES domain to the `listing-ses-notifications` topic. You can find it in `SES -> Domains -> (select your domain) -> Notifications`. Arn will be an output of `serverless deploy` command.

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
