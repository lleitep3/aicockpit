// Package agents is the assistant worker contract.
//
// Callers depend on Backend, not on a vendor CLI. Run accepts one task
// (prompt, working directory, timeout, optional model), streams events as
// the worker produces them, and returns the final result. Cancelling the
// context stops the worker. Errors are wrapped.
//
// CLIBackend is the local-process backend. It drives claude and codex
// through Adapter implementations. A Runner starts the process, so tests
// substitute fixture output and never invoke the real CLIs. This package
// does not add environment variables and does not read the vault. API
// backends and the cockpit ask command are later phases.
package agents
