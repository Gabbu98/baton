[user]: from the new gemini interaction update the Readme

[assistant]: Let me look at the recent changes and existing README to understand what needs updating.

[assistant]: Now I have a clear picture. The Gemini fix implemented actual context extraction — previously it was a no-op that just printed a message. The README needs to reflect the correct Gemini session file path and remove any implication that Gemini extraction was unsupported.

[assistant]: README updated — the Gemini session path now accurately reflects the `~/.gemini/tmp/<project>/chats/` lookup with its fallback, matching what `extractGeminiContext` actually does.

