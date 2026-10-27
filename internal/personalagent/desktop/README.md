# Cockpit Personal Agent Desktop

Electron shell for the existing local Cockpit Personal Agent service. The app
starts the bundled Go CLI, opens its loopback interface in an isolated desktop
window, and stops the service when the window closes.

## Local development

From this directory:

```sh
npm ci
npm test
npm start
```

`npm start` first builds the Go service for the current Linux architecture.
Use `npm start` from this checkout to test the current Personal Agent. A
previously installed `cockpit` binary may not include the `agent` command yet;
from the repository root, `go run . agent` runs the current source directly.

The composer accepts text files and PNG, JPEG, WebP, or GIF images. Paste a
screenshot with Ctrl+V, drag files onto the composer, or attach them from disk;
image turns are routed to the Codex adapter for multimodal input. Assistant
responses render Markdown, including code blocks, lists, and tables. The app
shows loading states while opening chats, reading attachments, and waiting for
agent responses.

The conversation list checks for changes every 15 seconds and marks updated
chats until opened. Click the bell in the top bar to opt in to system
notifications while the app is in the background. The Assistente Cockpit
session also accepts the same supported image and text attachments.

Use **Assistente Cockpit** in the sidebar for a separate operational assistant
session. The first version uses Codex structured tools to search conversation
titles/project IDs, read a bounded transcript, and propose a message to an
existing conversation. It never sends that message until you approve the exact
destination, selected model, and text shown in the proposal card. This feature
requires the Codex CLI to be installed and authenticated; other providers do
not yet expose structured Cockpit tools. The assistant uses a private,
read-only scratch workspace and automatically rejects Codex shell/file-change
approval requests.

## Linux installers

```sh
npm run dist:linux
```

The installer artifacts are written to `dist/` as an AppImage and a Debian
package. The build targets the current Linux architecture and requires Go,
Node.js, npm, and internet access for the first dependency download.

The app uses `~/.cockpit/workspace` as its default project workspace. Its
conversation history and routing configuration stay under
`~/.cockpit/conversations`. The Cockpit CLI provider and its authentication must
already be installed on the machine. The local service chooses an available
loopback port automatically; set `COCKPIT_AGENT_PORT` only when a fixed port is
required.
