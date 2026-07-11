package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Hex-4/bop/ai"
	"github.com/Hex-4/bop/config"
	"github.com/Hex-4/bop/scheduler"
	"github.com/Hex-4/bop/tools"

	"github.com/joho/godotenv"
)

func runServer() {
	userHome, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("error getting user home directory:", err)
		os.Exit(1)
	}

	defaultHome := filepath.Join(userHome, ".bop")
	configFile, err := config.Load(filepath.Join(defaultHome, "config.toml"))

	configFile.BopDir = defaultHome

	if err != nil {
		fmt.Println("error loading config:", err)
		os.Exit(1)
	}

	godotenv.Load(filepath.Join(defaultHome, ".env"))
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		fmt.Println("DISCORD_TOKEN not set")
		os.Exit(1)
	}

	/// set up composio ///
	composioSessionID, err := tools.CreateComposioSession()
	if err != nil {
		fmt.Println("error creating composio session:", err)
		os.Exit(1)
	}
	composioToolSchemas, err := tools.FetchComposioSchemas(composioSessionID)
	if err != nil {
		fmt.Println("error fetching composio schemas:", err)
		os.Exit(1)
	}

	externalToolsSlice, err := tools.NewComposioToolSlice(composioSessionID, composioToolSchemas)
	if err != nil {
		fmt.Println("error creating composio tool slice:", err)
		os.Exit(1)
	}

	agent := &ai.Agent{
		ActiveModel: configFile.Agent.Model,
		Config:      &configFile,
		Sessions:    make(map[string]*ai.Session),
		Tools:       tools.NewRegistry(filepath.Join(defaultHome, "workspace"), externalToolsSlice),
	}

	cronScheduler := scheduler.NewScheduler(agent)
	for _, tool := range cronScheduler.Tools() {
		agent.Tools[tool.Name] = tool
	}

	agent.ToolSchemas = tools.NewSchemaList(agent.Tools)

	for sessionID, sessionDescription := range configFile.Agent.SessionDescriptions {
		session := &ai.Session{
			ID:          sessionID,
			Description: sessionDescription,
			History:     make([]ai.Message, 0),
		}
		agent.Sessions[session.ID] = session
	}

	discordBot, err := NewDiscordBot(token, agent)
	if err != nil {
		fmt.Println("error creating discord bot:", err)
		os.Exit(1)
	}

	err = discordBot.Open()
	if err != nil {
		fmt.Println("error opening connection:", err)
		os.Exit(1)
	}

	cronScheduler.SendFunc = discordBot.Send
	cronScheduler.Cron.Start()
	fmt.Println("slopster is online 🫥")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	<-sc
	discordBot.Close()
	cronScheduler.Cron.Stop()
}
