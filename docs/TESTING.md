## Subscribing to a newsletter

`curl -v --data "newsletter=NewsletterName&email=foo@bar.com" https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev/subscribe`

## Email confirmation

Run previous "subscription" command to the real email address and click "Confirm email" button. Go to DynamoDB tables in AWS Console UI and check `listing-subscribers` table.

## Listing all subscribers

`curl -v -u :your-api-token https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev/subscribers?newsletter=NewsletterName`

## Importing subscribers

`curl -v -u :your-api-token -X PUT https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev/subscribers -H "Content-Type: application/json" --data-binary "@scripts/testsubscribers.json"`

## Handling bounce and complaint

Send email to `bounce@simulator.amazonses.com` from SES UI in AWS Console and check DynamoDB table `listing-sesnotify` if it contains the bounce notification.

Send email to `complaint@simulator.amazonses.com` from SES UI in AWS Console and check DynamoDB table `listing-sesnotify` if it contains the complaint notification. (this might not work as expected since it is dependent on your ISP as well)
