package email

import (
	"bytes"
	"log"
	"net/url"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/ribtoks/listing/pkg/common"
)

const (
	// Specify a configuration set. To use a configuration
	// set, comment the next line and line 92.
	//ConfigurationSet = "ConfigSet"

	// Subject line for the email
	Subject = "Confirm your email"

	// CharSet is the character encoding for the email
	CharSet = "UTF-8"

	// TextBody is a plain text copy of HTMLBody
	TextBody = `
Hello,

Thank you for subscribing to {{.Newsletter}} newsletter! Please confirm your email by clicking link below.

{{.ConfirmURL}}

You are receiveing this email because somebody, hopefully you, subscribed to {{.Newsletter}} newsletter. If it was not you, you can safely ignore this email.

{{.Newsletter}} team
`
)

var (
	HtmlTemplate *template.Template
	TextTemplate *template.Template
)

// SESMailer is an implementation of Mailer interface that works with AWS SES
type SESMailer struct {
	Sender string
	Secret string
	Svc    *ses.SES
}

var _ common.Mailer = (*SESMailer)(nil)

func (sm *SESMailer) confirmURL(newsletter, email string, confirmBaseURL string) (string, error) {
	token := common.Sign(sm.Secret, email)
	baseUrl, err := url.Parse(confirmBaseURL)
	if err != nil {
		log.Println("Malformed URL: ", err.Error())
		return "", err
	}
	params := url.Values{}
	params.Add(common.ParamNewsletter, newsletter)
	params.Add(common.ParamToken, token)
	baseUrl.RawQuery = params.Encode()
	return baseUrl.String(), nil
}

func (sm *SESMailer) sendEmail(email, htmlBody, textBody string) error {
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
					Data:    aws.String(htmlBody),
				},
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(textBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(Subject),
			},
		},
		Source: aws.String(sm.Sender),
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	result, err := sm.Svc.SendEmail(input)
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

func (sm *SESMailer) SendConfirmation(newsletter, email, name, confirmBaseURL string) error {
	confirmURL, err := sm.confirmURL(newsletter, email, confirmBaseURL)
	if err != nil {
		return err
	}

	nameParts := strings.Split(strings.TrimSpace(name), " ")

	data := struct {
		Newsletter string
		ConfirmURL string
		FirstName  string
	}{
		Newsletter: newsletter,
		ConfirmURL: confirmURL,
		FirstName:  strings.TrimSpace(nameParts[0]),
	}

	var htmlBodyTpl bytes.Buffer
	if err := HtmlTemplate.Execute(&htmlBodyTpl, data); err != nil {
		return err
	}

	var textBodyTpl bytes.Buffer
	if err := TextTemplate.Execute(&textBodyTpl, data); err != nil {
		return err
	}

	return sm.sendEmail(email, htmlBodyTpl.String(), textBodyTpl.String())
}

func init() {
	HtmlTemplate = template.Must(template.New("HtmlBody").Parse(HTMLBody))
	TextTemplate = template.Must(template.New("TextBody").Parse(TextBody))
}
