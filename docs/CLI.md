*Listing* serves couple of ordinary http endpoints and you can do all actions using `curl`, however, we provide `listing-cli` command line application that simplifies many actions and automates others.

## Options

Here are the supported options.

```
> ./listing-cli -help

  -auth-token string
    	Auth token for admin access
  -dry-run
    	Simulate selected action
  -email string
    	Email for subscribe|unsubscribe
  -format string
    	Ouput format of subscribers: csv|tsv|table|raw|yaml (default "table")
  -help
    	Print help
  -ignore-complaints
    	Ignore bounces and complaints for export
  -l string
    	Absolute path to log file (default "listing-cli.log")
  -mode string
    	Execution mode: subscribe|unsubscribe|export|import|delete
  -name string
    	(optional) Name for subscribe
  -newsletter string
    	Newsletter for subscribe|unsubscribe
  -no-unconfirmed
    	Do not export unconfirmed emails
  -no-unsubscribed
    	Do not export unsubscribed emails
  -secret string
    	Secret for email salt
  -stdout
    	Log to stdout and to logfile
  -url string
    	Base URL to the listing API
```

`secret` and `auth-token` are the parameters from `secrets.json` that you use when deploying lambda functions.

`listing-cli` supports `dry-run` parameter that allows to see what the specific chosen action or mode will do without actually doing it.

`url` parameter should be a "root" of your deployed API.

Use `-format yaml` in order to use *Listing* with [paperboy](https://github.com/rykov/paperboy) and `-format raw` for dump of the subscribers DB.

## Examples

```
# subscribing to a newsletter
./listing-cli -mode subscribe -email foo@bar.com -newsletter NewsletterName -url https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev

# listing all subscribers
./listing-cli -auth-token your-token-here -url "https://qwerty12345.execute-api.us-east-1.amazonaws.com/dev" -mode export -newsletter Listing1
```
