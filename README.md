# listing

[![Build Status](https://travis-ci.org/ribtoks/listing.svg?branch=master)](https://travis-ci.org/ribtoks/listing)
[![Build status](https://ci.appveyor.com/api/projects/status/ypmg5foasuiuf5lh/branch/master?svg=true)](https://ci.appveyor.com/project/Ribtoks/listing/branch/master)
[![codecov](https://codecov.io/gh/ribtoks/listing/branch/master/graph/badge.svg)](https://codecov.io/gh/ribtoks/listing)

[![Go Report Card](https://goreportcard.com/badge/github.com/ribtoks/listing)](https://goreportcard.com/report/github.com/ribtoks/listing)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/7ca0882c24314f01afe10bb857449ccb)](https://www.codacy.com/manual/ribtoks/listing?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=ribtoks/listing&amp;utm_campaign=Badge_Grade)
[![Maintainability](https://api.codeclimate.com/v1/badges/084d620cf3ef2f84ce99/maintainability)](https://codeclimate.com/github/ribtoks/listing/maintainability)

![license](https://img.shields.io/badge/license-MIT-blue.svg)
![copyright](https://img.shields.io/badge/%C2%A9-Taras_Kushnir-blue.svg)
![language](https://img.shields.io/badge/language-go-blue.svg)

## About

*Listing* is a small and simple service that allows to self-host email subscriptions list on AWS using Lambda, DynamoDB, SES and SNS. It handles email list subscribe/unsubscribe/confirm actions as well as email bounces and complaints (if you use same SES account for sending the newsletter).

### What problem it solves

Self-hosted email marketing solutions one can find today like [tinycampaign](https://github.com/parkerj/tinycampaign), [audience](https://github.com/aniftyco/audience), [mail-for-good](https://github.com/freeCodeCamp/mail-for-good), [colossus](https://github.com/vitorfs/colossus) and others have a few problems:

*   a need to run a webserver all of the time (resources are expensive)
*   operational overhead (certificate, updates, security)
*   they solve too many problems at the same time (subscription list, email sending, analytics, A/B testing, email templates)

There's also a [MoonMail](https://github.com/MoonMail/MoonMail) which is kind of a good approximation of a better system, but it is too complex to deploy and it tries to solve all problems at the same time as well. And there is almost no documentation since they are interested in selling SaaS version of it.

*Listing* is different. **It focuses only on building a subscription list.** It is simple, well documented, tested and easy to install.

As for other functionality, you can achieve analytics and A/B testing using systems like [Google Analytics](https://google.com/analytics) or others. Finding [good](https://github.com/InterNations/antwort) [email](https://github.com/leemunroe/responsive-html-email-template) [templates](https://github.com/mailgun/transactional-email-templates) or building ones using [available](http://mosaico.io/) [tools](https://beefree.io/) is also not a problem. And there are [many](https://github.com/rykov/paperboy) [ways](https://github.com/Circle-gg/thunder-mail) you can send those emails without a need to waste cloud computer resources at all other times.

### How does it work

*Listing* manages arbitrary amount of subscription lists using few lambda functions and a DynamoDB table.

*Listing* uses so-called "double-confirmation" system where user has to confirm it's email even after entering it on your website and pressing "Subscribe" button. The reason is that this is way more reliable in terms of loyal audience building.

Subscriptions are stored in a DynamoDB table and are managed by a lambda function with endpoints `/subscribe`, `/unsubscribe`, `/confirm` and `/subscribers`. The last endpoint is protected with BasicAuth for "admin" access (export/import of subscribers).

See also [detailed endpoints docs](https://github.com/ribtoks/listing/blob/master/docs/ENDPOINTS.md)

Bounces and Complaints are stored in additional DynamoDB table. This table is managed by another lambda function that is subscribed to SNS topic. AWS SES for a specific domain is a publisher to this SNS topic.

## How to use Listing

1.  Deploy *Listing* (described below)
2.  Test from the command line that everything works as expected (described below)
3.  Create html form for your website to subscribe to emails using *Listing* Lambda URL

Everything else (confirmation, unsubscribe) will be handled in the emails or automatically.

In order to operate *Listing* you can use `listing-cli` command line application that allows you to export and import subscribers, subscribe/unsubscribe single email and other actions.

See [full cli documentation](https://github.com/ribtoks/listing/blob/master/docs/CLI.md)

## How to avoid vendor lock

*Listing* is heavily dependent on AWS, but the size of DynamoDB table even with 100k subscribers is so small, you can safely export it to json using `/subscribers` endpoint and restore to any other cloud.

*Listing* uses serverless framework so migration to a different cloud will be less painful than from barebones AWS SAM templates.

## Deployment

See [full documentation](https://github.com/ribtoks/listing/blob/master/docs/DEPLOYMENT.md)

## Testing

See [full documentation](https://github.com/ribtoks/listing/blob/master/docs/TESTING.md)
