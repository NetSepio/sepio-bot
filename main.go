package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
	mux "github.com/vickyshaw29/discord-goBot/x/helper"
)

var Session, _ = discordgo.New()
var Router = mux.New()

func init() {
	Session.Token = "Bot " + os.Getenv("TOKEN")
	Session.AddHandler(Router.OnMessageCreate)
	Router.Route("/help", "Display this message.", Router.Help)
	Router.Route("*", "Send a quote", Router.Recommend)
}

func main() {
	// Session.Token = os.Getenv("TOKEN")
	// Session.Token = "Bot " + os.Getenv("TOKEN")
	err := Session.Open()
	if err != nil {
		log.Printf("error opening connection to Discord, %s\n", err)
		os.Exit(1)
	}

	// Wait for a CTRL-C
	log.Printf(`Now running. Press CTRL-C to exit.`)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	<-sc

	// Clean up
	Session.Close()
}
