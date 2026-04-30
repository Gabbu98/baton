# Baton 🪄

A lightweight context bridge for AI CLI sessions. Give each conversation a name and baton tracks it across sessions and AI tools. When a session ends, baton extracts the last few exchanges and saves them to a named chat file. On your next launch it automatically injects that context into `CLAUDE.md` / `GEMINI.md` (or `<CHATNAME>.md`) in your project directory — no copy-paste required.

```
  baton myproject claude
       │
       ▼
  ┌─────────────────────────────────────┐
  │  ~/Baton_Chats/myproject.md         │  ◄─ per-chat history (all sessions)
  └─────────────────────────────────────┘
       │ inject on next launch
       ▼
  ┌───────────────┐
  │ MYPROJECT.md  │  ◄─ claude/gemini reads this
  └───────────────┘
       │
       └──► ~/.config/baton/bridge.md           (latest session, always updated)
       └──► ~/Documents/Baton_Vault/handoff_log.md  (permanent log)
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

> **Note:** After setup, update the generated aliases in `~/.zshrc` to match the new two-argument format shown in [Aliases](#aliases-reference) below.

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

# Named chat: chat history keyed by project dir, writes CLAUDE.md / GEMINI.md
alias claude='baton "$(basename $PWD)" claude'
alias gemini='baton "$(basename $PWD)" gemini'

# Or fixed chat names that mirror the old single-file behaviour:
# alias claude='baton claude claude'
# alias gemini='baton gemini gemini'

alias baton='$HOME/.local/bin/baton'
alias baton-rebuild='(cd $HOME/.config/baton && go build -o $HOME/.local/bin/baton .) && echo "✅ baton rebuilt"'
alias bridge='glow $HOME/.config/baton/bridge.md'
```

Reload: `source ~/.zshrc`

### 3. Use it

```bash
# Start a session (chat name is "myproject", AI is claude)
baton myproject claude

# Continue the same chat with a different AI
baton myproject gemini

# Or just use the aliased form from within your project directory
claude          # expands to: baton "$(basename $PWD)" claude
```

---

## How It Works

### On launch

If `~/Baton_Chats/<chat-name>.md` has content from a previous session, baton writes it into `<CHATNAME>.md` in your current directory inside HTML comment markers:

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
1. Reads the most recent session JSONL from `~/.claude/projects/<project>/` (Claude) or the most recently modified session JSON from `~/.gemini/tmp/<project>/chats/` (Gemini, falls back to any session under `~/.gemini/tmp/*/chats/` if the project directory isn't found)
2. Extracts the last 6 meaningful turns (skipping tool calls, internal meta-messages)
3. Appends a timestamped entry to `~/Baton_Chats/<chat-name>.md`
4. Writes the same content to `~/.config/baton/bridge.md`
5. Appends a timestamped entry to `~/Documents/Baton_Vault/handoff_log.md`
6. Fires a macOS notification

---

## Configuration

All paths are configurable via environment variables. Only set these if your config lives somewhere non-standard (the setup script handles this automatically).

| Variable | Default | Purpose |
|----------|---------|---------|
| `BATON_CLAUDE_DIR` | `~/.claude` | Where Claude Code stores its config and history |
| `BATON_GEMINI_DIR` | `~/.gemini` | Where Gemini CLI stores its config |
| `BATON_BRIDGE_FILE` | `~/.config/baton/bridge.md` | The latest-session context file |
| `BATON_VAULT_DIR` | `~/Documents/Baton_Vault` | Permanent session log directory |
| `BATON_CHATS_DIR` | `~/Baton_Chats` | Per-chat session files |

Example for a non-standard Claude config location:

```bash
# ~/.zshrc
export BATON_CLAUDE_DIR="$HOME/Library/Application Support/claude"
```

### Why trace instead of move?

The Claude and Gemini CLIs hardcode their config directories. Moving those directories would break authentication, history, and settings for both CLIs. Baton instead reads from wherever those files already live and adapts — no disruption to the tools themselves.

---

## Aliases Reference

| Alias | Expands to | Description |
|-------|-----------|-------------|
| `claude` | `baton "$(basename $PWD)" claude` | Launch Claude with per-project context |
| `gemini` | `baton "$(basename $PWD)" gemini` | Launch Gemini with per-project context |
| `bridge` | `glow ~/.config/baton/bridge.md` | Preview latest session context |
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
└── bridge.md        — latest session context (auto-managed)

~/Baton_Chats/
├── myproject.md     — named chat history (all sessions, all AI tools)
└── ...

~/Documents/Baton_Vault/
└── handoff_log.md   — permanent timestamped log of all sessions
```

---

## Notes

- **Chat naming**: The chat name is used both to key the session file (`~/Baton_Chats/<name>.md`) and to determine the injected MD filename (`<NAME>.md`) in your project directory. Using the project directory name (e.g. `$(basename $PWD)`) gives you one chat file per project.
- **Cross-AI continuity**: Use the same chat name with different AI tools (`baton myproject claude`, then `baton myproject gemini`) to pass context between them.
- **CLAUDE.md scope**: `CLAUDE.md` (or `<CHATNAME>.md`) is read by Claude Code for the current project directory. The injected context is project-scoped, not global.
- **Private data**: Chat files, `bridge.md`, and `handoff_log.md` contain excerpts of your conversations. They are stored locally only.
