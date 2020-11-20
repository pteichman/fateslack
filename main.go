package main

import (
	"bufio"
	"flag"
	"log"
	"os"

	"github.com/pteichman/fate"
	"github.com/slack-go/slack"
)

func main() {
	flag.Parse()

	token := os.Getenv("SLACK_API_TOKEN")
	if token == "" {
		log.Fatalf("ERROR: Please set SLACK_API_TOKEN")
	}

	model := fate.NewModel(fate.Config{Stemmer: newStemmer("english")})
	for _, file := range flag.Args() {
		err := learnFile(model, file)
		if err != nil {
			log.Fatalf("ERROR: Learning %s: %s", file, err)
		}
	}

	rtm := slack.New(token).NewRTM()
	bot, err := newBot(rtm, model)
	if err != nil {
		log.Fatalf("ERROR: Starting bot: %s", err)
	}

	go rtm.ManageConnection()
	for {
		bot.handle(<-rtm.IncomingEvents)
	}
}

func learnFile(m *fate.Model, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	s := bufio.NewScanner(bufio.NewReader(f))
	for s.Scan() {
		m.Learn(s.Text())
	}

	return s.Err()
}
