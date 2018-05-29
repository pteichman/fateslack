package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/nlopes/slack"
	"github.com/pteichman/fate"
)

type Bot struct {
	RTM   *slack.RTM
	Model *fate.Model

	users []slack.User
}

func newBot(rtm *slack.RTM, model *fate.Model) (*Bot, error) {
	// TODO: don't fetch users at start, so new users are recognized.
	users, err := rtm.GetUsers()
	if err != nil {
		return nil, err
	}

	bot := &Bot{
		RTM:   rtm,
		Model: model,
		users: users,
	}

	return bot, nil
}

// toRe matches messages to users.
var toRe = regexp.MustCompile("@[^ ]+")

func (b *Bot) handle(e slack.RTMEvent) {
	info := b.RTM.GetInfo()

	switch ev := e.Data.(type) {
	case *slack.MessageEvent:
		if !(ev.Type == "message" && ev.SubType == "") {
			return
		}

		if ev.User == info.User.ID {
			// Don't ever learn anything from ourselves.
			return
		}

		text := strings.TrimSpace(cleanText(b.users, ev.Text))

		// Looking here for @mentions of any user. If the mention is at the beginning
		// of the message, strip it before continuing.
		m := toRe.FindStringSubmatch(text)
		if len(m) > 0 && strings.HasPrefix(text, m[0]) {
			text = strings.TrimSpace(strings.TrimPrefix(text, m[0]))
		}

		log.Printf("Learning: '%s'", text)
		b.Model.Learn(text)

		from := getUserByID(b.users, ev.User)
		if from != nil && from.IsBot {
			return
		}

		for _, mention := range m {
			if mention == "@"+info.User.Name {
				b.RTM.SendMessage(b.RTM.NewTypingMessage(ev.Channel))
				time.Sleep(time.Second / 2)

				reply := fate.QuoteFix(b.Model.Reply(text))
				log.Printf("Replying: '%s'", reply)

				msg := fmt.Sprintf("<@%s> %s", ev.User, reply)
				b.RTM.SendMessage(b.RTM.NewOutgoingMessage(msg, ev.Channel))
				break
			}
		}
	}
}

func getUserByID(users []slack.User, id string) *slack.User {
	for _, user := range users {
		if user.ID == id {
			return &user
		}
	}
	return nil
}

// Regexps for recovering what a user might have typed when Slack makes more
// full-featured text.
var (
	// chanRe is for mentioning channels like #foo.
	chanRe = regexp.MustCompile(`\x{003c}#.*?\|(.*?)\x{003e}`)
	// userRe is for mentioning users like @foo.
	userRe = regexp.MustCompile(`\x{003c}@(.*?)\x{003e}`)
	// linkRe is for mentioning http or https links.
	linkRe = regexp.MustCompile(`\x{003c}(https?://.*?)(\|.*)?\x{003e}`)
)

// cleanText attempts to recover what a user actually typed from text.
func cleanText(users []slack.User, text string) string {
	text = chanRe.ReplaceAllString(text, "$1")

	text = userRe.ReplaceAllStringFunc(text, func(match string) string {
		m := userRe.FindStringSubmatch(match)
		if user := getUserByID(users, m[1]); user != nil {
			return "@" + user.Name
		}
		return "@unknown"
	})

	text = linkRe.ReplaceAllString(text, "$1")

	return text
}
