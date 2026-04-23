<!-- baton:context:start -->
## Baton Context Bridge
_Carried over: 2026-04-23 09:56_

[user]: add a README.md with instruction how to set it up on a mac for now with all requirement and add a automated setup script according to my existing config if the user has claude and gemini files in different place see whether its best to move the or trace them and adapt

[assistant]: Let me gather all the details about the current setup before writing anything.

[assistant]: Now I have the full picture. Let me update `main.go` to support env var overrides for config paths, then write both files.

[assistant]: Now write the setup script:

[assistant]: Now the README:

[assistant]: Done. Here's what was added/changed:

**`main.go`** — added four env var overrides at the top (`BATON_CLAUDE_DIR`, `BATON_GEMINI_DIR`, `BATON_BRIDGE_FILE`, `BATON_VAULT_DIR`) so non-standard config paths can be set without modifying source.

**`setup.sh`** — automated macOS setup that:
1. Checks/installs prerequisites: Homebrew, Go, glow (optional prompt)
2. Verifies Claude and Gemini CLIs are present (warns with install links if not)
3. Detects config directories — checks `BATON_*_DIR` env vars first, then the CLI's own env hints (`CLAUDE_CONFIG_DIR`, `GEMINI_HOME`), then defaults (`~/.claude`, `~/.gemini`), then prompts interactively if nothing found
4. Writes `BATON_*_DIR` exports to `~/.zshrc` only if the path is non-standard — no-op for standard setups
5. Builds and installs the binary
6. Adds all aliases idempotently (won't duplicate if run again)

**`README.md`** — covers requirements, quick setup, manual setup, how the bridge works, config env vars, alias reference, and the rationale for tracing vs moving (short answer: moving breaks the CLIs since they hardcode their config dirs).
<!-- baton:context:end -->
