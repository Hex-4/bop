package tools

func newSendMessage(send func(string) error) Tool {
	return Tool{
		Name:        "send_message",
		Description: "Send a message to the user. IMPORTANT: THIS IS THE ONLY WAY TO COMMUNICATE WITH THE USER. The agent harness you are running in does NOT send your final response text to the user. To talk to the user, you MUST use this tool. This allows for a better user experience, as before kicking off a task or going and using other tools, you can acknowledge the user's request (with a simple response such as \"On it!\", make some tool calls, and respond with your final message. Again, you MUST use THIS TOOL and only THIS TOOL to talk to the user.",
		Parameters: map[string]Parameter{
			"text": {
				Type:        "string",
				Description: "The message text to send to the user.",
				Required:    true,
			},
		},
		Execute: func(args map[string]any) (string, error) {
			text := ArgString(args, "text")
			err := send(text)
			if err != nil {
				return "Error: " + err.Error(), nil
			}
			return "Message sent successfully", nil
		},
	}
}
