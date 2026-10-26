// Package hindsight manages an optional local Hindsight server.
//
// The server is the official standalone image
// (ghcr.io/vectorize-io/hindsight), published on host ports 8888 (API) and
// 9999 (control plane) unless the operator overrides the host ports. The
// container uses embedded pg0: HINDSIGHT_API_DATABASE_URL is never set, and
// the data directory /home/hindsight/.pg0 is a named Docker volume.
//
// LLM API keys and any other secrets are read from the cockpit vault when a
// container is created and passed only as container environment variables.
// This package does not write those values to disk or logs. Docker is reached
// through Runtime so tests can substitute FakeRuntime and never start a daemon.
//
// This package does not implement memory.Memory and does not talk to the
// knowledge base. Retain, Recall, and Reflect stay in internal/memory.
package hindsight
