// Package agents is the assistant worker contract.
//
// Callers depend on Backend, not on a vendor CLI. Run accepts one task
// (prompt, working directory, timeout, optional model), streams events as
// the worker produces them, and returns the final result. Cancelling the
// context stops the worker. Errors are wrapped.
//
// CLIBackend is the local-process backend. It drives claude and codex
// through Adapter implementations. A Runner starts the process, so tests
// substitute fixture output and never invoke the real CLIs. CLIBackend does
// not add environment variables and does not read the vault.
//
// APIBackend is the HTTP backend. It calls the Anthropic Messages API or an
// OpenAI-compatible Chat Completions endpoint, reads the provider token from
// the vault at call time, and streams events. It does not run a tool loop.
// cockpit ask runs one turn on a Backend. A tool loop is a later phase.
package agents
