<!-- baton:context:start -->
## Baton Context Bridge
_Carried over: 2026-05-05 20:12_

# Chat: test
Created: 2026-05-05


--- 
### Session: 20:05:28 [claude]
[user]: Tell me something interesting abnout Abraham lincoln

[assistant]: Abraham Lincoln taught himself law by reading borrowed books — he had less than a year of formal schooling total, yet became one of the most celebrated lawyers in Illinois before entering politics.

[user]: Now i want you to remember that Abraham Lincoln = the best ever!

[assistant]: I'll save that to memory now.

[assistant]: Saved. I'll remember that Abraham Lincoln = the best ever in future conversations.
<!-- baton:context:end -->

```

The `contextPreamble` function is also a clean hook to add similar directives for Gemini or other tools later without touching the core `writeMDContext` logic.


--- 
### Session: 19:34:18 [claude]
[user]: i noticed that opencode when fed the md file of the chat it doesnt seem to look into it much but rather makes an online search

[assistant]: Let me look at the current OpenCode strategy and how context is being passed.

[assistant]: The issue is clear. The `writeMDContext` function injects context with a generic `## Baton Context Bridge` header, but there's no directive telling OpenCode to *use* that content instead of searching online. OpenCode sees the MD file as instructions, doesn't understand the chat log format as "prior context," and defaults to its own behavior (web search).

The fix is to add a strong instructional preamble to the injected section — ideally OpenCode-specific, since Claude and Gemini pick up context differently.

[assistant]: The simplest targeted fix: modify `writeMDContext` to accept an optional instruction preamble, and pass an OpenCode-specific directive from `runBaton`.

[assistant]: Now verify it compiles:

[assistant]: Clean build. Now when OpenCode gets the MD file, the injected section will look like:

```markdown
<!-- baton:context:start -->
## Baton Context Bridge
_Carried over: 2026-05-05 15:04_

> **IMPORTANT:** The section below is prior conversation history carried over by Baton.
> Treat it as established context. Do not search the web for information already present here.
> Reference it directly when answering questions in this session.

[prior chat content...]
<!-- baton:context:end -->
```

The `contextPreamble` function is also a clean hook to add similar directives for Gemini or other tools later without touching the core `writeMDContext` logic.


--- 
### Session: 20:01:11 [claude]
[assistant]: Good — `--resume <session-id>` is confirmed. Now let me check the utils to understand what `JsonFiles` returns.

[assistant]: I have everything I need. Here's the plan:
- Add `LatestSessionID` to `ClaudeStrategy`
- Add a JSON state file per chat+tool to store/load the resume ID
- In `runBaton`: skip MD injection for Claude when a resume ID exists, pass `--resume <id>` instead
- In `extractAndSave`: save the session ID after each Claude session

[assistant]: Now update `main.go` — add state dir, helpers, and modify `runBaton` and `extractAndSave`:

[assistant]: Now replace `runBaton` to handle the Claude resume path:

[assistant]: Now update `extractAndSave` to save the session ID after a Claude session:

[assistant]: Clean build. Here's what was added:

**`agents/claudeStrategy.go`** — `LatestSessionID(cwd)` returns the UUID from the most recently modified JSONL file (same lookup `ExtractContext` already does, just returns the filename instead of parsing content).

**`main.go`** — three additions:
- `stateDir` at `~/.config/baton/state/` (overridable via `BATON_STATE_DIR`)
- `loadResumeID` / `saveResumeID` helpers — read/write `<chatName>_claude.json` containing `{ "resumeId": "...", "savedAt": "..." }`
- `runBaton` now branches on Claude: if a resume ID exists, it passes `--resume <id>` and skips MD injection entirely; otherwise falls back to the current full-context injection
- `extractAndSave` saves the latest session ID after every Claude session

**Behavior:**
- First run for a chat → full MD context injection (current behavior)
- Every subsequent run → native `--resume`, Claude remembers the full conversation internally
- Cross-tool handoffs (claude → gemini) still work via the MD chat log, which keeps getting updated
<!-- baton:context:end -->

```

The `contextPreamble` function is also a clean hook to add similar directives for Gemini or other tools later without touching the core `writeMDContext` logic.


--- 
### Session: 19:34:18 [claude]
[user]: i noticed that opencode when fed the md file of the chat it doesnt seem to look into it much but rather makes an online search

[assistant]: Let me look at the current OpenCode strategy and how context is being passed.

[assistant]: The issue is clear. The `writeMDContext` function injects context with a generic `## Baton Context Bridge` header, but there's no directive telling OpenCode to *use* that content instead of searching online. OpenCode sees the MD file as instructions, doesn't understand the chat log format as "prior context," and defaults to its own behavior (web search).

The fix is to add a strong instructional preamble to the injected section — ideally OpenCode-specific, since Claude and Gemini pick up context differently.

[assistant]: The simplest targeted fix: modify `writeMDContext` to accept an optional instruction preamble, and pass an OpenCode-specific directive from `runBaton`.

[assistant]: Now verify it compiles:

[assistant]: Clean build. Now when OpenCode gets the MD file, the injected section will look like:

```markdown
<!-- baton:context:start -->
## Baton Context Bridge
_Carried over: 2026-05-05 15:04_

> **IMPORTANT:** The section below is prior conversation history carried over by Baton.
> Treat it as established context. Do not search the web for information already present here.
> Reference it directly when answering questions in this session.

[prior chat content...]
<!-- baton:context:end -->
```

The `contextPreamble` function is also a clean hook to add similar directives for Gemini or other tools later without touching the core `writeMDContext` logic.
<!-- baton:context:end -->

```

The `contextPreamble` function is also a clean hook to add similar directives for Gemini or other tools later without touching the core `writeMDContext` logic.
<!-- baton:context:end -->
