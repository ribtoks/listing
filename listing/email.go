package main

import (
	"bytes"
	"log"
	"net/url"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ses"
)

const (
	// Specify a configuration set. To use a configuration
	// set, comment the next line and line 92.
	//ConfigurationSet = "ConfigSet"

	// The subject line for the email.
	Subject = "Confirm your email"

	// The character encoding for the email.
	CharSet = "UTF-8"

	TextBody = `
Hello,

Thank you for subscribing to {{.Newsletter}} newsletter! Please confirm your email by clicking link below.

{{.ConfirmUrl}}

You are receiveing this email because somebody, hopefully you, subscribed to {{.Newsletter}} newsletter. If it was not you, you can safely ignore this email.

Xpiks team
`
)

type Mailer interface {
	SendConfirmation(newsletter, email string, confirmUrl string) error
}

type SESMailer struct {
	sender string
	secret string
	svc    *ses.SES
}

func (sm *SESMailer) confirmUrl(newsletter, email string, confirmBaseUrl string) (string, error) {
	token := Sign(sm.secret, email)
	baseUrl, err := url.Parse(confirmBaseUrl)
	if err != nil {
		log.Println("Malformed URL: ", err.Error())
		return "", err
	}
	params := url.Values{}
	params.Add(paramNewsletter, newsletter)
	params.Add(paramToken, token)
	baseUrl.RawQuery = params.Encode()
	return baseUrl.String(), nil
}

func (sm *SESMailer) SendConfirmation(newsletter, email string, confirmBaseUrl string) error {
	confirmUrl, err := sm.confirmUrl(newsletter, email, confirmBaseUrl)
	if err != nil {
		return err
	}
	data := struct {
		Newsletter string
		ConfirmUrl string
	}{
		Newsletter: newsletter,
		ConfirmUrl: confirmUrl,
	}

	var htmlBodyTpl bytes.Buffer
	if err := HtmlTemplate.Execute(&htmlBodyTpl, data); err != nil {
		return err
	}

	var textBodyTpl bytes.Buffer
	if err := TextTemplate.Execute(&textBodyTpl, data); err != nil {
		return err
	}

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: []*string{
				aws.String(email),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(htmlBodyTpl.String()),
				},
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(textBodyTpl.String()),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(Subject),
			},
		},
		Source: aws.String(sm.sender),
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	result, err := sm.svc.SendEmail(input)
	log.Printf("Email send result=%v", result)

	// Display error messages if they occur.
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				log.Println(ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				log.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				log.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}

		return err
	}

	return nil
}
