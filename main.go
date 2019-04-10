package main

import (
	"time"
	"log"
//	"fmt"
//	"strconv"
//	"net/url"
	"os"

	tb "gopkg.in/tucnak/telebot.v2"
)

// Google Sis Bot
func newsBot() {
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("SECRET_TOKEN"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle(tb.OnText, func(m *tb.Message) {
		b.Send(m.Sender, "This bot does not currently supported the interactive mode")

	})

	channel, channelGetErr := b.ChatByID(os.Getenv("CHANNEL_ID"))

	if channelGetErr != nil {
	    log.Fatal(channelGetErr)
	    return
	}

	b.Send(channel, "This test message")

	b.Start()
}

func main() {
	go newsBot()	
	select {}
}

