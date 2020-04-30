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

## Configure custom domain

If you want to deploy _listing_ as `listing.yourdomain.com` you will need to do couple of things:

*   Request custom certificate in AWS Certificate manager for region `us-east-1` (even if you plan to deploy to other regions) for your domain
*   Make sure `secrets.json` from previous step has correct domain(s) configured
*   Run `STAGE=dev REGION=eu-west-1 make create_domain` (replace stage and region with your preferences) - this has to be run only once per "lifetime"

You can see those domains in API Gateway section of AWS Console. Currently _listing_ assumes you are using `dev-listing.yourdomain.com` for `dev` and `listing.yourdomain.com` for `prod`. You can change this logic in `serverless-api.yml` file in `customDomains` section.

## Deploy listing

`STAGE=dev REGION=eu-west-1 make deploy-all`

This command will compile and deploy whole _listing_ to AWS in the said stage and region. In case you are making changes to API or to Admin parts, you can redeploy them separately from DB. Use commands `make deploy-db`, `make deploy-api` and `make deploy-admin` and their `make remove-xxx` counterparts to deploy and remove separate pieces.

## Configure confirm and redirect URLs

Go to AWS Console UI and in Lambda section find `listing-subscribe` function. Set `CONFIRM_URL` in it's environmental variables to point to the API Gateway address for `listing-confirm` function's address.

(optional - you can do that in the end) Configure `SUBSCRIBE_REDIRECT_URL`, `UNSUBSCRIBE_REDIRECT_URL`, `CONFIRM_REDIRECT_URL` in the appropriate lambda function to point to the pages on your website.

## Configure SNS topic for bounces and complaints

In order to exit "sandbox" mode in AWS SES you need to have a procedure for handling bounces and complaints. *Listing* provides this functionality, but you have to do 1 manual action.

Go to [AWS Console UI and set Bounce and Complaint](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/configure-sns-notifications.html) SNS topic's ARN for your SES domain to the `listing-ses-notifications` topic. You can find it in `SES -> Domains -> (select your domain) -> Notifications`. Arn will be an output of `serverless deploy` command for `serverless-db.yml` config. Example of such ARN: `arn:aws:sns:eu-west-1:1234567890:dev-listing-ses-notifications`.

## Configure redirect URLs to your website

Open lambda function properties in the AWS management console and set all `_REDIRECT_URL` variables to the appropriate values that point to your website. You will need to have pages for email confirmation, unsubscribe confirmation etc.

