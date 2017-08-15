package main

import (
	r "gopkg.in/gorethink/gorethink.v3"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"strings"
	"os"
	"net/http"
)

var (
	bot     *tgbotapi.BotAPI
	session *r.Session
)

func main() {
	var err error

	token := os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("TOKEN env variable not specified!")
	}

	bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	//bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Webhooks example (don't forget to change URL!)
	_, err = bot.SetWebhook(tgbotapi.NewWebhookWithCert("https://krasovsky.me:8443/"+bot.Token, "certs/cert.pem"))
	if err != nil {
		log.Fatal(err)
	}

	go http.ListenAndServeTLS(":8443", "certs/cert.pem", "certs/key.pem", nil)
	updates := bot.ListenForWebhook("/" + bot.Token)

	// Long-polling
	/*
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	*/

	go InitConnectionPool()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if strings.HasPrefix(update.Message.Text, "/start") {
			go StartCommand(update)
		}

		if strings.HasPrefix(update.Message.Text, "/help") {
			go HelpCommand(update)
		}

		if strings.HasPrefix(update.Message.Text, "/save") {
			go SaveCommand(update)
		}

		if strings.HasPrefix(update.Message.Text, "/full") {
			go FullCommand(update)
		}

		if strings.HasPrefix(update.Message.Text, "/today") {
			go TodayCommand(update)
		}

		if strings.HasPrefix(update.Message.Text, "/tomorrow") {
			go TomorrowCommand(update)
		}

		if strings.HasPrefix(update.Message.Text, "/get") {
			go GetCommand(update)
		}

		if strings.HasPrefix(update.Message.Text, "/status") {
			go StatusCommand(update)
		}

		if strings.HasPrefix(update.Message.Text, "/update") {
			go UpdateCommand(update)
		}

		if strings.HasPrefix(update.Message.Text, "/delete") {
			go DeleteCommand(update)
		}
	}
}
