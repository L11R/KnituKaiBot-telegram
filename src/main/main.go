package main

import (
	r "gopkg.in/gorethink/gorethink.v3"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"net/http"
	"strings"
)

const (
	dbUrl = "krasovsky.me"
)

var (
	bot     *tgbotapi.BotAPI
	session *r.Session
)

func main() {
	var err error

	bot, err = tgbotapi.NewBotAPI("406656319:AAGOLg46Oj0ogKPqNEZaRozj1NI9x_ptB3s")
	if err != nil {
		log.Panic(err)
	}

	//bot.Debug = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	_, err = bot.SetWebhook(tgbotapi.NewWebhookWithCert("https://krasovsky.me:8443/"+bot.Token, "cert.pem"))
	if err != nil {
		log.Fatal(err)
	}

	go http.ListenAndServeTLS(":8443", "cert.pem", "key.pem", nil)
	updates := bot.ListenForWebhook("/" + bot.Token)

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
