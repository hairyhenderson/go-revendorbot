/*
Example commandline Go program that processes the input given by github-responder
*/

package main

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	revendorbot "github.com/hairyhenderson/go-revendorbot"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("incorrect usage: call as %s <eventType> <deliveryID> (args were %v)", os.Args[0], os.Args)
	}
	eventType := os.Args[1]
	deliveryID := os.Args[2]

	payload, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal("failed to read stdin", err)
	}

	ctx := context.Background()
	bot, err := revendorbot.New(ctx)
	if err != nil {
		log.Fatal(err)
	}

	err = bot.Handle(eventType, deliveryID, payload)
	if err != nil {
		log.Fatal(err)
	}
}
