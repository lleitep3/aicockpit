// Package ask runs one assistant turn.
//
// A turn recalls memory, builds a prompt, runs one agents.Backend, streams
// the answer, and retains a short record. It does not loop on tools and it
// does not route by task. Callers inject memory and a Catalog, so tests use
// fakes and do not start a CLI or open a network connection.
package ask
