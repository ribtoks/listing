`listing-send` is a simple application that allows to send Go-templated emails to subscribers through SMTP server (AWS SES).

`listing-send` requires path to html and text templates. These templates are executed using common parameters (`-params` option, available in template as `{{.Params.xxx}}`) and recepient params from the subscription list (`-list` option, available in template as `{{.Recepient.xxx}}`).

After execution, rendered emails can be saved locally using `-dry-run` option (they are saved to directory from parameter `-out`) or sent to SMTP server.

## Options

```
> ./listing-send -help
  -dry-run
    	Simulate selected action
  -from-email string
    	Sender address
  -from-name string
    	Sender name
  -help
    	Print help
  -html-template string
    	Path to html email template
  -l string
    	Absolute path to log file (default "listing-send.log")
  -list string
    	Path to file with email list (default "list.json")
  -out string
    	Path to directory for dry run results (default "./")
  -params string
    	Path to file with common params (default "params.json")
  -pass string
    	SMTP password flag
  -rate int
    	Emails per second sending rate (default 25)
  -stdout
    	Log to stdout and to logfile
  -subject string
    	Html campaign subject
  -txt-template string
    	Path to text email template
  -url string
    	SMTP server url
  -user string
    	SMTP username flag
  -workers int
    	Number of workers to send emails (default 2)
```

## Example

```
listing-send -url "smtp://email-smtp.us-east-1.amazonaws.com:587" -user "" -pass "" -subject "Newsletter digest January 2020" -from-email "email@example.com" -from-name "Email from Example" -html-template layouts/layout-1-2-3.html -txt-template layouts/_default.text -params content/newsletters/jan-2020-params.json -list lists/test.json
```
