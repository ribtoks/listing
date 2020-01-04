## Subscribing to a newsletter

`./listing-cli -mode subscribe -email foo@bar.com -newsletter NewsletterName -url https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev`

OR

`curl -v --data "newsletter=NewsletterName&email=foo@bar.com" https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev/subscribe`

## Email confirmation

Run previous "subscription" command to the real email address and click "Confirm email" button. Go to DynamoDB tables in AWS Console UI and check `listing-subscribers` table.

## Unsubscribing from a newsletter

`./listing-cli -mode unsubscribe -email foo@bar.com -secret your-secret -newsletter NewsletterName -url https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev`

Unsubscribing with curl might be a little bit tricky since you need to generate a token which is handled in `listing-cli` for you.

## Listing all subscribers

`./listing-cli -auth-token your-token-here -url "https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev" -mode export -newsletter Listing1`

OR

`curl -v -u :your-api-token https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev/subscribers?newsletter=NewsletterName`

## Importing subscribers

`cat scripts/add_subscribers.json | ./listing-cli/listing-cli -auth-token your-api-token -mode import -url https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev`

OR

`curl -v -u :your-api-token -X PUT https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev/subscribers -H "Content-Type: application/json" --data-binary "@scripts/add_subscribers.json"`

## Deleting subscribers

`curl -v -u :your-api-token -X DELETE https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev/subscribers -H "Content-Type: application/json" --data-binary "@scripts/remove_subscribers.json"`

## Handling bounce and complaint

Send email to `bounce@simulator.amazonses.com` from SES UI in AWS Console and check DynamoDB table `listing-sesnotify` if it contains the bounce notification.

Send email to `complaint@simulator.amazonses.com` from SES UI in AWS Console and check DynamoDB table `listing-sesnotify` if it contains the complaint notification. (this might not work as expected since it is dependent on your ISP as well)
