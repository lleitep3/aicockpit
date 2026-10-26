// Package memory is the assistant memory contract.
//
// Callers depend on Memory, not on a storage engine. Retain stores one item,
// Recall searches items, and Reflect summarizes the matches. Every method
// takes a context, returns wrapped errors, and does no network I/O.
//
// KB is the default backend. It reads and writes the local knowledge base
// through internal/kb. InMemory is a process-local backend. Fake is a test
// double with the same behavior plus injected errors and recorded calls.
//
// A Hindsight backend can implement Memory in a later package. This package
// does not import a Hindsight client. Optional process lifecycle for a local
// Hindsight server lives in internal/hindsight. Map the fields as follows: Query.Bank
// is the bank id, Query.Text is the recall or reflect query, Item.Content is
// the retained text, Item.Context is the situational context, Item.DocumentID
// is the source document id, Item.Metadata is the string metadata, and
// Item.Timestamp is when the fact was observed. Reflect on that backend
// returns the service's synthesized text; a backend that cannot synthesize
// returns ErrReflectNotSupported.
package memory
