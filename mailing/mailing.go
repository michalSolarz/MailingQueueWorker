package mailing

import (
	"github.com/adjust/redismq"
	"encoding/json"
	"gitlab.com/michalSolarz/AuthAPI/authorization"
	"github.com/Sirupsen/logrus"
	"gopkg.in/mailgun/mailgun-go.v1"
	"fmt"
	"html/template"
	"bytes"
)

type Email struct {
	Subject   string
	Template  string
	Plaintext string
}

type Mailer struct {
	Logger    *logrus.Logger
	Templates map[string]Email
	Sender    mailgun.Mailgun
	From      string
}

var handledTemplates = map[string]string{
	authorization.AccountActivationTokenType: authorization.AccountActivationTokenType,
	authorization.PasswordResetTokenType:     authorization.PasswordResetTokenType,
}

func ProceedMailingToken(mailer *Mailer, message *redismq.Package) {
	token := authorization.MailingToken{}
	err := json.Unmarshal([]byte(message.Payload), &token)
	if err != nil {
		panic(err)
	}
	if _, ok := handledTemplates[token.TokenType]; ok {
		emailTemplate := mailer.Templates[token.TokenType]
		htmlTemplate, err := template.New("html").Parse(emailTemplate.Template)
		plaintextTemplate, err := template.New("plaintext").Parse(emailTemplate.Plaintext)
		if err != nil {
			mailer.Logger.Error(err)
		}
		var htmlTpl bytes.Buffer
		if err := htmlTemplate.Execute(&htmlTpl, token); err != nil {
			mailer.Logger.Error(err)
		}
		var plaintextTpl bytes.Buffer
		if err := plaintextTemplate.Execute(&plaintextTpl, token); err != nil {
			mailer.Logger.Error(err)
		}

		email := mailer.Sender.NewMessage(mailer.From, emailTemplate.Subject, plaintextTpl.String(), token.Email)
		email.SetHtml(htmlTpl.String())

		resp, id, err := mailer.Sender.Send(email)
		if err != nil {
			mailer.Logger.Fatal(err)
		}
		mailer.Logger.Info(fmt.Sprintf("Mail: %s request, with response: %s", id, resp))
	} else {
		mailer.Logger.Info(fmt.Sprintf("Unhandled email type: %s", token.TokenType))
	}
}
