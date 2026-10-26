// Package memoryhindsight is the optional Hindsight backend for memory.Memory.
//
// Retain, Recall, and Reflect call the official Hindsight Go client. Tests
// substitute Client with Fake and do not open a network connection. The
// integration is off until memory.hindsight.enabled is set. Search and index
// keep working against the local knowledge base when Hindsight is disabled,
// misconfigured, or unreachable.
//
// Two banks are used: one for the project and one for the user. Names can be
// set in memory.hindsight.banks or derived as project-<name> and user-<name>.
// kb search recalls both banks and appends the hits to the local results.
// kb index retains documents into the project bank.
//
// internal/providers declares the same two banks as streamable HTTP MCP
// servers when memory.hindsight.enabled is set. The endpoint is
// <api>/mcp/<bank>/ on the API server.
package memoryhindsight
