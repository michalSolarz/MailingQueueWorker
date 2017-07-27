package main

import (
	"github.com/adjust/redismq"
	"github.com/Sirupsen/logrus"
	"runtime"
	"gitlab.com/michalSolarz/MailingQueueWorker/mailing"
	"io/ioutil"
	"fmt"
	"gitlab.com/michalSolarz/AuthAPI/authorization"
	"gopkg.in/mailgun/mailgun-go.v1"
)

const (
	REDIS_HOST         = "localhost"
	REDIS_PORT         = "6379"
	REDIS_PASSWORD     = ""
	QUEUE_DB           = 10
	QUEUE_NAME         = "mailingQueue"
	SERVER_PORT        = "9999"
	MAILING_DOMAIN     = ""
	MAILING_API_KEY    = ""
	MAILING_PUBLIC_KEY = ""
	MAILING_EMAIL_FROM = "no-replay@" + MAILING_DOMAIN
)

var log = logrus.New()

func main() {
	runtime.GOMAXPROCS(5)
	server := redismq.NewServer(REDIS_HOST, REDIS_PORT, REDIS_PASSWORD, QUEUE_DB, SERVER_PORT)
	server.Start()
	queue := redismq.CreateQueue(REDIS_HOST, REDIS_PORT, REDIS_PASSWORD, QUEUE_DB, QUEUE_NAME)
	templates := map[int]string{
		0: authorization.AccountActivationTokenType,
		1: authorization.PasswordResetTokenType,
	}
	go readQueue(queue, "1", &mailing.Mailer{log, loadEmails(templates), mailgun.NewMailgun(MAILING_DOMAIN, MAILING_API_KEY, MAILING_PUBLIC_KEY), MAILING_EMAIL_FROM})
	select {}
}

func readQueue(queue *redismq.Queue, prefix string, mailer *mailing.Mailer) {
	consumer, err := queue.AddConsumer("mailing" + prefix)
	if err != nil {
		panic(err)
	}
	consumer.ResetWorking()
	for {
		p, err := consumer.Get()
		if err != nil {
			log.Error(err)
			continue
		}
		mailing.ProceedMailingToken(mailer, p)
		err = p.Ack()
	}
}

func loadEmails(templates map[int]string) map[string]mailing.Email {
	emailsCollection := map[string]mailing.Email{}
	for _, template := range templates {
		emailsCollection[template] = loadEmail(template)
	}
	return emailsCollection
}

func loadEmail(name string) mailing.Email {
	subject, err := ioutil.ReadFile(fmt.Sprintf("subjects/%s", name))
	checkError(err)
	template, err := ioutil.ReadFile(fmt.Sprintf("templates/%s.html", name))
	checkError(err)
	plaintext, err := ioutil.ReadFile(fmt.Sprintf("plaintext/%s", name))
	checkError(err)
	return mailing.Email{Subject: string(subject), Template: string(template), Plaintext: string(plaintext)}
}

func checkError(e error) {
	if e != nil {
		panic(e)
	}
}
