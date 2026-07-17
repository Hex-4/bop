# Bop v1 Roadmap

## what's done
- [x] discord bot connects, receives messages, responds
- [x] openrouter integration (single model, stateless HTTP calls)
- [x] agent struct with per-session conversation history
- [x] typing indicator while thinking
- [x] bot ignores itself, responds to DMs + @mentions
- [x] config package (TOML parsing, go:embed defaults)
- [x] workspace + context file injection (configurable via `context_files`)
- [x] tool calling loop (detect tool_calls → execute → feed results → loop)
- [x] max-iterations safeguard
- [x] `read_file` / `write_file` tools
- [x] `shell` tool (timeout, working directory, blocklist)
- [x] `web_search` / `web_fetch` / `web_highlights` tools
- [x] composio integration (dynamic external tools via composio API)
- [x] cron + one-shot scheduling (`create_cron`, `schedule_once`, `remove_job`, `list_jobs`)
- [x] `cli/init.go` first-run setup

## phase R: trigger/sink refactor

the big architecture change. bop moves from "agent owns everything" to a clean three-primitive split: agents (brains), triggers (fire the agent, own sessions), sinks (dumb output destinations). `send_message` replaces the native response + status handler. no more `StatusHandler`.

### step 1: the contract (DONE)
- [x] `triggers/` package with `Sink` interface (`Send(text string) error`)
- [x] `SessionStore` in `triggers/` (in-memory, mutex-protected)
- [x] `Agent.Ask(messages []Message, toolList map[string]tools.Tool) ([]Message, error)`
  - agent no longer owns sessions or mutates input
  - returns new messages to append (slice pass-by-value safe)
  - tool loop rebuilds messages each iteration
- [x] `Agent.SystemPrompt() string` — static agent identity (reads context files fresh)
- [x] `send_message` tool factory in `tools/sink.go`

### step 2: discord trigger
- [x] refactor `handleMessage` into a trigger that owns a `SessionStore`
- [x] trigger assembles full messages slice: `agent.SystemPrompt()` + dynamic context (time, session desc) + session history + user message
- [x] call `Ask`, append `newMessages` to session, save to store
- [x] keep typing indicator, mention/DM filter, error-posting fallback
- [x] temporarily keep `DiscordBot.Send(sessionID, message)` alive for cron (killed in step 3)
- [x] fix `append(newMessages, messages...)` → `append(messages, newMessages...)` in Ask

### step 3: cron trigger
- [ ] cron gets its own `SessionStore` keyed by job ID
- [ ] each job builds its own `DiscordSink` for its target channel
- [ ] per-fire registry includes `send_message` (unless silent)
- [ ] remove `DiscordBot.Send` and `cronScheduler.SendFunc` — cron uses sinks now
- [ ] silent jobs just don't add `send_message` to the registry

### step 4: verify + cleanup
- [ ] smoke test: discord messages, mid-turn `send_message`, cron fires, session persistence
- [ ] stupid-agent reminder: trigger inspects `newMessages` for `send_message` usage, injects system note if missing
- [ ] kill dead code: `Agent.Sessions`, `Session` struct, old `assembleSystemPrompt`
- [ ] fix `read_file`/`write_file` error convention (return content strings, not Go errors)
- [ ] path validation for `read_file`/`write_file` (block `..` traversal)
- [ ] add stupid agent reminder
- - [ ] update `AGENTS.md` (architecture section is way out of date)
- [ ] update `config.toml` — remove `tools_footer`

## phase 5: agents.toml + subagents

bop moves from one agent in config to N agents, each with its own model, system prompt, tools.

- [ ] `agents.toml` parser + loader
- [ ] each agent: model, system_prompt_file, context_files, default tools
- [ ] triggers wired in `server.go`: `<trigger, agent, sink>` triples
- [ ] `dispatch_subagent` tool — main agent can spawn a subagent and wait for its response
- [ ] subagent's "return value" is a `kill(message)` that injects into the parent's history
- [ ] subagents are fresh-session per dispatch (no retention between calls)
- [ ] subagents write to files if they need to remember things

## phase 6: librarian (auto-compaction)

replaces the old `/compact` slash command. no manual compaction — the librarian fires automatically when idle + context is full.

- [ ] librarian agent definition (in `agents.toml`)
  - system prompt: "you are a librarian, extract wisdom from conversations"
  - tools: `read_file`, `write_file` (to MEMORY/OPEN), no `send_message` (or a narrating one)
  - fresh session per invocation
- [ ] idle + context threshold trigger (configurable interval + threshold)
  - default idle: ~5-10 minutes
  - default threshold: ~60-70% of model context window
- [ ] librarian receives the full conversation history as input
- [ ] writes bullet points to `MEMORY.md` (facts, preferences — low churn)
- [ ] writes active follow-ups to `OPEN.md` (high churn, get checked off)
- [ ] trims MEMORY/OPEN if they get too long (deduplication for MEMORY, completion for OPEN)
- [ ] MEMORY/OPEN are in the main agent's `context_files` (picked up automatically)
- [ ] main agent's old history is cleared/trimmed after librarian runs

## phase 7: composio events + more triggers

- [ ] composio event triggers (fire on external events, not just cron/messages)
- [ ] events can target any agent (not just main)
- [ ] `ssh` tool
- [ ] `switch_model` tool

## phase 8: slash commands + polish

- [ ] `/reset` — clear session history
- [ ] `/model` — switch model for current session
- [ ] discord message chunking (2000 char limit)
- [ ] error handling hardening (nil choices, empty responses, rate limits)
- [ ] graceful shutdown (finish in-flight requests via `sync.WaitGroup`)
