package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"gopkg.in/telegram-bot-api.v4"
)

func main() {
	var err error
	var server *http.Server
	var stopServer = false

	botToken := flag.String("token", "", "telegram bot token")
	debug := flag.Bool("debug", true, "turn debug bode on/off")
	webhookBaseURL := flag.String("webhookBaseURL", "", "Base URL for webhook")
	port := flag.String("port", "80", "port to listen")
	flag.Parse()

	if *botToken == "" {
		fmt.Printf("usage:\n hashnodebot -token <token>\n")
		os.Exit(1)
	}

	bot, err := tgbotapi.NewBotAPI(*botToken)
	if err != nil {
		log.Fatalf("Initializing bot with the given token failed : %v\n", err)
	}
	bot.Debug = *debug
	log.Printf("Intialized with username %s\n", bot.Self.UserName)

	var updates tgbotapi.UpdatesChannel

	if *webhookBaseURL != "" {
		err := startWithWebHook(bot, *webhookBaseURL)
		if err != nil {
			log.Fatal(err)
		}
		updates = bot.ListenForWebhook("/" + bot.Token)
		server, err = startServer(*port)
		if err != nil {
			log.Fatal(err)
		}

	} else {
		config := startWithPolling(bot, 1)
		updates, err = bot.GetUpdatesChan(*config)
		if err != nil {
			log.Fatal(err)
		}

	}

	go exitGracefully(func(done chan bool) {
		if server != nil {
			server.Shutdown(context.Background())
		}
		done <- true
		return
	})

	for update := range updates {
		fmt.Println(10)
		if update.Message == nil {
			continue
		}
		if stopServer {
			break
		}

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)
		msg.ReplyToMessageID = update.Message.MessageID

		bot.Send(msg)
	}
}

func startWithWebHook(bot *tgbotapi.BotAPI, webhookURL string) error {
	_, err := bot.SetWebhook(tgbotapi.NewWebhook(webhookURL + "/" + bot.Token))
	if err != nil {
		return err
	}
	info, err := bot.GetWebhookInfo()
	if err != nil {
		return err
	}

	if info.LastErrorDate != 0 {
		return fmt.Errorf("Callback to telegram failed: %s ", info.LastErrorMessage)
	}
	return nil
}

func startWithPolling(bot *tgbotapi.BotAPI, timeout int) *tgbotapi.UpdateConfig {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = timeout
	return &u
}

func startServer(port string) (*http.Server, error) {
	url := "0.0.0.0:" + port
	listener, err := net.Listen("tcp", url)
	if err != nil {
		return nil, err
	}

	server := &http.Server{
		ReadTimeout:    time.Second * 30,
		WriteTimeout:   time.Second * 60,
		MaxHeaderBytes: 1 << 20,
	}

	go server.Serve(listener)
	return server, nil

}

func exitGracefully(handleShutdown func(chan bool)) {
	signalChan := make(chan os.Signal, 1)
	cleanUpDone := make(chan bool)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		sig := <-signalChan
		fmt.Printf("\nRecived %v\n", sig)
		// handle cleanup tasks
		handleShutdown(cleanUpDone)

		fmt.Printf("Cleanup Completed...Now shutting down\n")
		os.Exit(0)
	}()
	<-cleanUpDone
}
