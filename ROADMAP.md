(generated with Amp)

# Bop v1 Roadmap

## what's done
- [x] discord bot connects, receives messages, responds
- [x] openrouter integration (single model, stateless HTTP calls)
- [x] agent struct with per-session conversation history
- [x] typing indicator while thinking
- [x] bot ignores itself, responds to DMs + @mentions
- [x] config package stubbed (TOML parsing)

## phase 1: config-driven agent
- [x] wire TOML config into main (load from `~/.bop/config.toml`)
- [x] move model, system prompt, provider into config
- [x] session descriptions in config (`[agent.sessions]` map)
- [x] inject session description into system prompt per-channel
- [x] create `~/.bop/` dir + default config on first run

## phase 2: workspace + memory
- [x] create `~/.bop/workspace/` on startup
- [ ] seed `memory.md` and `pending.md` on first run
  - [ ] write default `memory.md` (empty template with headers for facts/preferences)
  - [ ] write default `pending.md` (empty template for ongoing threads/tasks)
  - [ ] only seed if files don't already exist (don't clobber)
- [x] inject workspace files as context (configurable via `context_files`)
  - [x] read each file listed in `config.Agent.ContextFiles` on every message
  - [x] prepend file contents to system prompt (or as separate system messages)
  - [x] handle missing files gracefully (warn in logs, don't crash)

## phase 3: tool calling loop
- [x] define tool schema format (match openrouter/openai function calling spec)
  - [x] add `tools` field to `ChatRequest` struct (array of tool definitions)
  - [x] add `tool_calls` field to response `Message` struct
  - [x] define a `Tool` interface or struct: name, description, parameters schema, execute func
- [x] agent loop: model responds → check for tool calls → execute → feed results back → loop until final text response
  - [x] detect `tool_calls` in response (vs plain text content)
  - [x] look up tool by name, validate params
  - [x] execute tool, capture result as string
  - [x] append tool result as a `role: "tool"` message with `tool_call_id`
  - [x] re-call the model with updated messages
  - [x] loop until response has no tool calls (just text)
  - [x] add a max-iterations safeguard (prevent infinite loops)
- [x] stream progress updates to discord as tools run
  - [x] send/edit a discord message with current tool name + status
  - [x] update the message as each tool completes
- [x] `read_file` and `write_file` as first tools
  - [x] `read_file`: takes `path` param, reads from `~/.bop/workspace/<path>`, returns contents
  - [x] `write_file`: takes `path` + `content`, writes to workspace, returns confirmation
  - [ ] path validation: block `..` traversal, enforce workspace scope

## phase 4: more tools
- [x] `shell` tool (sandboxed, block dangerous commands)
  - [x] execute command via `os/exec`, capture stdout+stderr
  - [x] blocklist of dangerous commands/patterns (`rm -rf`, `sudo`, `curl | sh`, etc.)
  - [x] timeout per command (e.g. 30s)
  - [x] working directory locked to workspace
- [x] `web_search` tool
  - [x] pick a search API (SearXNG, Brave Search, Google Custom Search)
  - [x] return top N results as title + snippet + URL
- [ ] `ssh` tool
  - [ ] connect to a host defined in config (don't let the agent pick arbitrary hosts)
  - [ ] run a command, return stdout+stderr
  - [ ] timeout + output length cap
- [ ] `switch_model` tool (agent picks its own model mid-conversation)
  - [ ] update `agent.Model` for the current session
  - [ ] optionally restrict to an allowlist of models in config

## phase 5: cron + scheduling
- [ ] `create_cron` tool (agent schedules recurring jobs)
  - [ ] parse cron expression (use `robfig/cron` library)
  - [ ] each job stores: cron expr, prompt to send, target session, optional model override
  - [ ] register job with a cron scheduler running in the background
- [ ] `schedule_once` tool (one-time future tasks)
  - [ ] stores: timestamp, prompt, target session, optional model override
  - [ ] use `time.AfterFunc` or a ticker loop to fire at the right time
- [ ] per-job model override (cheap model for heartbeats, smart model for reflection)
  - [ ] temporarily swap `agent.Model` for the duration of the job's execution
- [ ] persist schedules to disk, restore on restart
  - [ ] save jobs to `~/.bop/schedules.json` (or TOML)
  - [ ] on startup, load and re-register all persisted jobs
  - [ ] remove one-time jobs after they fire
- [ ] default evening reflection cron
  - [ ] seed a default cron job on first run (e.g. daily at 9pm)
  - [ ] prompt tells agent to review memory.md + pending.md and update them

## phase 6: slash commands + polish
- [ ] `/reset` — clear session history
  - [ ] register discord slash command on startup
  - [ ] handler: wipe `session.History`, confirm to user
- [ ] `/compact` — summarize history into a shorter context
  - [ ] send current history to model with "summarize this conversation" prompt
  - [ ] replace history with a single system message containing the summary
- [ ] `/model` — switch model for current session
  - [ ] takes model name as argument
  - [ ] validate against available models or allowlist
- [ ] discord message chunking (2000 char limit)
  - [ ] split long responses into multiple messages at newline boundaries
  - [ ] preserve code blocks across splits (don't break mid-block)
- [ ] error handling hardening (nil choices, empty responses, rate limits)
  - [ ] check `len(result.Choices) == 0` before indexing
  - [ ] handle HTTP 429 (rate limit) with backoff/retry
  - [ ] handle non-200 responses with readable error messages
- [ ] graceful shutdown (finish in-flight requests)
  - [ ] on SIGINT/SIGTERM, stop accepting new messages
  - [ ] wait for in-flight `ask()` calls to complete (use `sync.WaitGroup`)
  - [ ] then close discord connection
