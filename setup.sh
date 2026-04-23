#!/usr/bin/env bash
# Baton setup script for macOS

set -euo pipefail

BATON_SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="$HOME/.local/bin"
VAULT_DIR="$HOME/Documents/Baton_Vault"

GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; CYAN='\033[0;36m'; NC='\033[0m'
ok()   { echo -e "${GREEN}✅  $1${NC}"; }
warn() { echo -e "${YELLOW}⚠️   $1${NC}"; }
err()  { echo -e "${RED}❌  $1${NC}"; exit 1; }
info() { echo -e "    $1"; }
hdr()  { echo -e "\n${CYAN}$1${NC}"; }

echo ""
echo "🪄  Baton — Context Bridge Setup"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# ── 1. OS ────────────────────────────────────────────────────────────────────
[[ "$(uname)" == "Darwin" ]] || err "This script is for macOS only."
ok "macOS $(sw_vers -productVersion)"

# ── 2. Homebrew ───────────────────────────────────────────────────────────────
hdr "Checking prerequisites..."
if ! command -v brew &>/dev/null; then
    err "Homebrew is required but not found.\nInstall it from https://brew.sh then re-run this script."
fi
ok "Homebrew $(brew --version | head -1 | awk '{print $2}')"

# ── 3. Go ─────────────────────────────────────────────────────────────────────
if ! command -v go &>/dev/null; then
    info "Go not found — installing via Homebrew..."
    brew install go
fi
ok "Go $(go version | awk '{print $3}')"

# ── 4. Claude CLI ─────────────────────────────────────────────────────────────
# Use 'type -P' to find the real binary, bypassing any shell aliases
CLAUDE_BIN=$(type -P claude 2>/dev/null || true)
if [[ -z "$CLAUDE_BIN" ]]; then
    warn "Claude CLI not found in PATH."
    info "Install it from: https://claude.ai/code"
    CLAUDE_OK=false
else
    ok "Claude CLI → $CLAUDE_BIN"
    CLAUDE_OK=true
fi

# ── 5. Gemini CLI ─────────────────────────────────────────────────────────────
GEMINI_BIN=$(type -P gemini 2>/dev/null || true)
if [[ -z "$GEMINI_BIN" ]]; then
    warn "Gemini CLI not found in PATH."
    info "Install it from: https://ai.google.dev/gemini-api/docs/gemini-cli"
    GEMINI_OK=false
else
    ok "Gemini CLI → $GEMINI_BIN"
    GEMINI_OK=true
fi

# ── 6. glow (optional — used by the 'bridge' alias) ─────────────────────────
if ! command -v glow &>/dev/null; then
    echo ""
    read -rp "   Install glow (Markdown renderer for the 'bridge' alias)? [Y/n] " _ans
    if [[ "${_ans,,}" != "n" ]]; then
        brew install glow && ok "glow installed"
    else
        warn "Skipping glow. The 'bridge' alias will show raw Markdown instead."
    fi
else
    ok "glow $(glow --version 2>/dev/null | head -1)"
fi

# ── 7. Detect config directories ─────────────────────────────────────────────
# Strategy: trace (read from where the CLIs put their files) rather than move,
# because moving breaks the CLIs themselves.
# Non-standard locations are stored in BATON_*_DIR env vars written to ~/.zshrc.

hdr "Detecting config directories..."

BATON_CLAUDE_DIR_CUSTOM=""
BATON_GEMINI_DIR_CUSTOM=""

detect_config_dir() {
    local label="$1"
    local default_path="$2"
    local env_hint="$3"      # env var the CLI itself might honor
    local result=""

    # 1. Honour existing BATON_*_DIR overrides from the current env
    local baton_var="BATON_${label^^}_DIR"
    if [[ -n "${!baton_var:-}" && -d "${!baton_var}" ]]; then
        result="${!baton_var}"
        ok "$label config: $result  (from \$$baton_var)"
        echo "$result"; return
    fi

    # 2. Check if the CLI's own env var points somewhere
    if [[ -n "${!env_hint:-}" && -d "${!env_hint}" ]]; then
        result="${!env_hint}"
        warn "$label config found via \$$env_hint → $result"
        info "Baton will trace that path (no files moved)."
        echo "$result"; return
    fi

    # 3. Default location
    if [[ -d "$default_path" ]]; then
        result="$default_path"
        ok "$label config: $result  (default)"
        echo "$result"; return
    fi

    # 4. Interactive fallback
    warn "$label config not found at $default_path."
    read -rp "    Enter the path to your $label config directory (or press Enter to skip): " _custom
    if [[ -n "$_custom" && -d "$_custom" ]]; then
        result="$_custom"
        ok "$label config: $result  (custom)"
    else
        warn "$label history extraction will be disabled until the CLI is used."
    fi
    echo "$result"
}

DETECTED_CLAUDE=$(detect_config_dir "CLAUDE" "$HOME/.claude" "CLAUDE_CONFIG_DIR")
DETECTED_GEMINI=$(detect_config_dir "GEMINI" "$HOME/.gemini" "GEMINI_HOME")

[[ "$DETECTED_CLAUDE" != "$HOME/.claude" && -n "$DETECTED_CLAUDE" ]] && \
    BATON_CLAUDE_DIR_CUSTOM="$DETECTED_CLAUDE"
[[ "$DETECTED_GEMINI" != "$HOME/.gemini" && -n "$DETECTED_GEMINI" ]] && \
    BATON_GEMINI_DIR_CUSTOM="$DETECTED_GEMINI"

# ── 8. Build binary ───────────────────────────────────────────────────────────
hdr "Building baton..."
mkdir -p "$INSTALL_DIR"
(cd "$BATON_SRC" && go build -o "$INSTALL_DIR/baton" .) || err "Build failed. Check Go errors above."
ok "Binary installed → $INSTALL_DIR/baton"

# ── 9. Vault directory ────────────────────────────────────────────────────────
mkdir -p "$VAULT_DIR"
ok "Vault directory → $VAULT_DIR"

# ── 10. Shell configuration ───────────────────────────────────────────────────
hdr "Configuring shell..."

# Detect shell config file
if [[ "$SHELL" == */zsh ]]; then
    SHELL_RC="$HOME/.zshrc"
elif [[ "$SHELL" == */bash ]]; then
    SHELL_RC="$HOME/.bash_profile"
    [[ -f "$HOME/.bashrc" ]] && SHELL_RC="$HOME/.bashrc"
else
    warn "Unknown shell ($SHELL). Defaulting to ~/.zshrc"
    SHELL_RC="$HOME/.zshrc"
fi
info "Shell config: $SHELL_RC"

touch "$SHELL_RC"

# Idempotent line appender
append_if_missing() {
    local line="$1"
    local grep_key="${2:-$line}"
    if ! grep -qF "$grep_key" "$SHELL_RC" 2>/dev/null; then
        echo "$line" >> "$SHELL_RC"
        info "Added:   $line"
    else
        info "Exists:  $line"
    fi
}

# PATH
append_if_missing 'export PATH="$HOME/.local/bin:$PATH"' 'local/bin:$PATH'

# BATON_*_DIR only if non-standard
if [[ -n "$BATON_CLAUDE_DIR_CUSTOM" ]]; then
    append_if_missing "export BATON_CLAUDE_DIR=\"$BATON_CLAUDE_DIR_CUSTOM\"" "BATON_CLAUDE_DIR"
fi
if [[ -n "$BATON_GEMINI_DIR_CUSTOM" ]]; then
    append_if_missing "export BATON_GEMINI_DIR=\"$BATON_GEMINI_DIR_CUSTOM\"" "BATON_GEMINI_DIR"
fi

# Aliases — write as a block if none exist yet
if ! grep -qF "alias baton=" "$SHELL_RC" 2>/dev/null; then
    cat >> "$SHELL_RC" << 'ALIASES'

# Baton — AI context bridge
alias baton='$HOME/.local/bin/baton'
alias baton-rebuild='(cd $HOME/.config/baton && go build -o $HOME/.local/bin/baton .) && echo "✅ baton rebuilt"'
alias claude='baton claude'
alias gemini='baton gemini'
alias bridge='glow $HOME/.config/baton/bridge.md'
ALIASES
    ok "Aliases added to $SHELL_RC"
else
    ok "Aliases already present in $SHELL_RC"
fi

# ── Done ──────────────────────────────────────────────────────────────────────
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}🎉  Setup complete!${NC}"
echo ""
echo "   Reload your shell:"
echo -e "   ${CYAN}source $SHELL_RC${NC}"
echo ""
echo "   Then just type ${CYAN}claude${NC} or ${CYAN}gemini${NC} as usual."
echo "   After each session, baton saves context to ${CYAN}bridge.md${NC}."
echo "   Next launch it auto-injects it into ${CYAN}CLAUDE.md${NC} / ${CYAN}GEMINI.md${NC}."
echo "   Run ${CYAN}bridge${NC} to preview the current context."
echo ""
