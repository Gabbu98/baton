# Baton 🪄

A lightweight context bridge for AI CLI sessions. When you finish a `claude` or `gemini` session, baton extracts the last few exchanges and saves them to `bridge.md`. On your next session it automatically injects that context into `CLAUDE.md` / `GEMINI.md` in your project directory so the AI picks up where you left off — no copy-paste required.

```
  ┌─────────┐  exit   ┌────────────┐  next launch  ┌───────────┐
  │  claude │ ──────► │ bridge.md  │ ────────────►  │ CLAUDE.md │ ◄─ claude reads this
  └─────────┘         └────────────┘                └───────────┘
                            │
                            └──► Documents/Baton_Vault/handoff_log.md  (permanent log)
```

## Requirements

| Tool | Version | Install |
|------|---------|---------|
| macOS | 13+ | — |
| [Homebrew](https://brew.sh) | any | `/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"` |
| Go | 1.21+ | `brew install go` |
| [Claude Code CLI](https://claude.ai/code) | any | see link |
| [Gemini CLI](https://ai.google.dev/gemini-api/docs/gemini-cli) | any | `npm install -g @google/gemini-cli` |
| [glow](https://github.com/charmbracelet/glow) *(optional)* | any | `brew install glow` |

`glow` is only needed for the `bridge` alias that renders the context file as formatted Markdown.

---

## Quick Setup

```bash
git clone <this-repo> ~/.config/baton
bash ~/.config/baton/setup.sh
source ~/.zshrc
```

The script handles everything: builds the binary, installs it, detects config directories, and adds shell aliases.

---

## Manual Setup

### 1. Build and install the binary

```bash
cd ~/.config/baton
go build -o ~/.local/bin/baton .
```

Make sure `~/.local/bin` is on your `PATH`:

```bash
# ~/.zshrc
export PATH="$HOME/.local/bin:$PATH"
```

### 2. Add shell aliases

```bash
# ~/.zshrc
alias baton='$HOME/.local/bin/baton'
alias baton-rebuild='(cd $HOME/.config/baton && go build -o $HOME/.local/bin/baton .) && echo "✅ baton rebuilt"'
alias claude='baton claude'
alias gemini='baton gemini'
alias bridge='glow $HOME/.config/baton/bridge.md'
```

Reload: `source ~/.zshrc`

### 3. Use it

Just type `claude` or `gemini` as you normally would. Baton wraps the CLI transparently.

---

## How It Works

### On launch
If `bridge.md` has content from a previous session, baton writes it into `CLAUDE.md` (or `GEMINI.md`) in your current directory inside HTML comment markers:

```markdown
<!-- baton:context:start -->
## Baton Context Bridge
_Carried over: 2026-04-23 09:48_

[user]: tell me smt interesting
[assistant]: The Unix timestamp `1111111111` occurred on March 18, 2005...
<!-- baton:context:end -->

...rest of your CLAUDE.md...
```

The marker block is replaced on each subsequent launch — existing file content below it is preserved.

### On exit
After the CLI process ends, baton:
1. Reads the most recent session JSONL from `~/.claude/projects/<project>/` (Claude) or falls back gracefully (Gemini — no parseable history available yet)
2. Extracts the last 6 meaningful turns (skipping tool calls, internal meta-messages)
3. Writes them to `~/.config/baton/bridge.md`
4. Appends a timestamped entry to `~/Documents/Baton_Vault/handoff_log.md`
5. Fires a macOS notification

---

## Configuration

All paths are configurable via environment variables. Only set these if your config lives somewhere non-standard (the setup script handles this automatically).

| Variable | Default | Purpose |
|----------|---------|---------|
| `BATON_CLAUDE_DIR` | `~/.claude` | Where Claude Code stores its config and history |
| `BATON_GEMINI_DIR` | `~/.gemini` | Where Gemini CLI stores its config |
| `BATON_BRIDGE_FILE` | `~/.config/baton/bridge.md` | The context handoff file |
| `BATON_VAULT_DIR` | `~/Documents/Baton_Vault` | Permanent session log directory |

Example for a non-standard Claude config location:

```bash
# ~/.zshrc
export BATON_CLAUDE_DIR="$HOME/Library/Application Support/claude"
```

### Why trace instead of move?

The Claude and Gemini CLIs hardcode their config directories. Moving those directories would break authentication, history, and settings for both CLIs. Baton instead reads from wherever those files already live and adapts — no disruption to the tools themselves.

---

## Aliases Reference

| Alias | Command | Description |
|-------|---------|-------------|
| `claude` | `baton claude` | Launch Claude Code with context bridge |
| `gemini` | `baton gemini` | Launch Gemini CLI with context bridge |
| `bridge` | `glow ~/.config/baton/bridge.md` | Preview current context |
| `baton-rebuild` | `go build ...` | Rebuild binary after source changes |

You can pass any flags through: `claude --resume`, `gemini --model gemini-2.5-pro`, etc.

---

## Updating

After pulling changes or editing `main.go`:

```bash
baton-rebuild
```

---

## File Layout

```
~/.config/baton/
├── main.go          — source
├── go.mod
├── setup.sh         — automated setup
├── README.md
└── bridge.md        — current context (auto-managed)

~/Documents/Baton_Vault/
└── handoff_log.md   — permanent timestamped log of all sessions
```

---

## Notes

- **Gemini history**: The Gemini CLI does not currently write parseable chat history to disk. Baton skips extraction after Gemini sessions and leaves `bridge.md` unchanged. Context injected into `GEMINI.md` on launch still works (from a prior Claude session or a manually edited `bridge.md`).
- **CLAUDE.md scope**: `CLAUDE.md` is read by Claude Code for the current project directory. The injected context is project-scoped, not global.
- **Private data**: `bridge.md` and `handoff_log.md` contain excerpts of your conversations. They are stored locally only.
