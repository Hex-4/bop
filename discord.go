package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/Hex-4/bramble/ai"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	dg    *discordgo.Session
	agent *ai.Agent
}

func NewDiscordBot(token string, agent *ai.Agent) (*DiscordBot, error) {
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	d := &DiscordBot{
		dg:    dg,
		agent: agent,
	}
	dg.AddHandler(d.handleMessage)
	dg.Identify.Intents = discordgo.IntentsAll
	return d, nil
}

func (d *DiscordBot) Open() error {
	return d.dg.Open()
}

func (d *DiscordBot) Close() error {
	return d.dg.Close()
}

func (d *DiscordBot) Send(sessionID string, message string) {
	channelID := strings.TrimPrefix(sessionID, "discord:")
	d.dg.ChannelMessageSend(channelID, message)
}

func (d *DiscordBot) handleMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	isDM := m.GuildID == ""

	if !isDM {
		mentioned := false
		for _, user := range m.Mentions {
			if user.ID == s.State.User.ID {
				mentioned = true
				break
			}
		}
		if !mentioned {
			return
		}
	}
	messageText := m.Message.Content
	messageText = "Discord message from user id " + m.Author.ID + ": " + messageText
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				s.ChannelTyping(m.ChannelID)
				time.Sleep(10 * time.Second)
			}
		}
	}()

	statusHandler := d.newStatusHandler(m.ChannelID)

	aiResponse, err := d.agent.Ask("discord:"+m.ChannelID, messageText, statusHandler)

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "something broke. slopster is sorry. here's the error: "+err.Error())
		return
	}
	footer := statusHandler.Footer()
	if footer != "" {
		s.ChannelMessageSend(m.ChannelID, aiResponse+"\n-# "+footer)
	} else {
		s.ChannelMessageSend(m.ChannelID, aiResponse)
	}
	done <- true
}

func (d *DiscordBot) newStatusHandler(channelID string) *ai.StatusHandler {
	var statusMsgID string
	var completedEmojis []string

	return &ai.StatusHandler{
		OnToolStart: func(emoji string, detail string) {
			line := "-# " + strings.Join(completedEmojis, "") + " ⟡ " + emoji + " " + detail

			if statusMsgID == "" {
				msg, _ := d.dg.ChannelMessageSend(channelID, line)
				statusMsgID = msg.ID
			} else {
				d.dg.ChannelMessageEdit(channelID, statusMsgID, line)
			}

			completedEmojis = append(completedEmojis, emoji)
		},
		OnDone: func() {
			if statusMsgID != "" {
				d.dg.ChannelMessageDelete(channelID, statusMsgID)
			}
		},
		Footer: func() string {
			if completedEmojis == nil {
				return ""
			}
			return strings.Join(completedEmojis, "") + " " + strconv.Itoa(len(completedEmojis)) + " tools"
		},
	}
}
