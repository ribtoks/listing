# listing
Service to manage marketing email list subscriptions

## Deployment

### Generic prerequisites

*   Buy/Own a custom domain
*   Create an AWS account
*   Configure and verify your domain in AWS SES (Simple Email Service)
*   Exit "sandbox" mode in SES (you have to contact support)
*   Create "confirm"/"unsubscribed" pages on your website

### Install prerequisites:

*   Go
*   Node.js (for serverless)
*   Serverless (`npm install -g serverless`)
*   Serverless plugins (`npm install` in repository root)
*   Configure AWS credentials

### Edit secrets

`cp secrets.json.example secrets.json`

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
