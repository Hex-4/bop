# AGENTS.md

## about bop
bop is a lightweight AI agent framework in Go. it runs as a persistent daemon, connects to Discord, and lets an LLM (via OpenRouter) use tools, manage its own memory, and schedule its own tasks. single binary, TOML config, flat markdown workspace.

## about the developer
hex4 is a middle schooler learning Go while building this. **teach Go concepts and show examples rather than just writing code.** explain *why*, not just *what*. when writing code, keep it simple — no over-engineering, no premature abstractions.

## project structure
```
main.go          — CLI router (os.Args), dispatches to runServer() or cli.RunInit()
server.go        — wires config → agent → discord, signal handling
discord.go       — DiscordBot struct, message handler, status updates
ai/openrouter.go — Agent struct, tool loop, OpenRouter API calls
config/config.go — TOML config loading, go:embed defaults
tools/tools.go   — Tool struct, registry, schema generation
tools/files.go   — read_file, write_file tools
tools/shell.go   — shell tool (os/exec)
cli/init.go      — bop init (first-run setup)
```

## conventions
- **package main** for server/discord/CLI code, separate packages for `ai`, `config`, `tools`, `cli`
- all tool definitions follow the same pattern: `newToolName(workspaceDir string) Tool` returning a closure
- tools are registered in `NewRegistry()` in `tools/tools.go`
- tool schemas are built via `Tool.ToSchema()` returning `map[string]any` (OpenRouter JSON format)
- config lives at `~/.bop/config.toml`, workspace at `~/.bop/workspace/`
- agent communicates status to Discord via `StatusHandler` callbacks (not direct Discord imports)
- `Message` struct is shared across user/assistant/tool roles, using `omitempty` for optional fields (except `Content`)
- errors from tools are returned as content strings to the model, not Go errors (so the model can adapt)

## important patterns
- **no circular imports** — `ai` imports `tools` and `config`, never the other way
- **closures for tool state** — tools capture `workspaceDir` via closures, no globals
- **pointer receivers** on Agent and DiscordBot (they have mutable state)
- **`&` for pointers**, config is passed as `*config.Config` to avoid copying

## do NOT
- write or edit files unless explicitly asked — explain and teach first, let hex4 write the code
- add dependencies without checking `go.mod` first
- put Discord-specific code in the `ai` package
- use `interface{}` — use `any` (same thing, modern syntax)
- add `omitempty` to `Message.Content` (breaks StepFun provider)
- ignore errors from `callModel` or `json.Unmarshal`

## build & run
```
go run .                    # won't do anything (needs subcommand)
go run . server run         # start the bot
go run . init               # first-run setup
find . -name '*.go' | entr -r go run . server run   # dev mode with auto-restart
```

## roadmap
see `ROADMAP.md` for current progress. currently working through phase 3 (tool calling loop) and phase 4 (more tools).
