Here are endpoints created by *Listing*

Endpoint | Method | Parameters | Description
--- | --- | --- | ---
`/subscribe` | POST | `newsletter`, `email`, `name`? | Subscribe form on your website
`/confirm` | GET | `newsletter`, `token` | "Confirm Email" button in the confirmation email
`/unsubscribe` | GET | `newsletter`, `token` | "Unsubscribe" link in the newsletter emails
`/subscribers` | GET | `newsletter` | Protected API to retrieve all subscribers for a newsletter
`/subscribers` | PUT | JSON with Subscribers array | Protected API to import subscribers

`token` parameter is a salted hash of the email used to uniquely identify every user. It is a security measure to protect from unauthorized unsubscribes/confirmations.

`name` parameter in `/subscribe` endpoint is optional.
