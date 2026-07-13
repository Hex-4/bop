package discord

import (
	"strings"
	"time"

	"github.com/Hex-4/bop/ai"
	"github.com/bwmarrin/discordgo"
)

type DiscordBot struct {
	dg    *discordgo.Session
	agent *ai.Agent
}

type DiscordSink struct {
	dg        *discordgo.Session
	channelID string
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

	aiResponse, err := d.agent.Ask("discord:"+m.ChannelID, messageText)

	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "something broke. slopster is sorry. here's the error: "+err.Error())
		return
	}
	s.ChannelMessageSend(m.ChannelID, aiResponse)
	done <- true
}
