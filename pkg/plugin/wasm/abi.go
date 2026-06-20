// Package wasm documents the Herald WASM provider ABI.
//
// Guest exports:
//   - malloc(size uint32) uint32
//   - Send(inPtr, inLen uint32) (outPtr, outLen uint32)
//   - ValidateConfig(inPtr, inLen uint32) (outPtr, outLen uint32) — optional
//
// Send input JSON:
//   { "config": {...}, "request": {...}, "context": {...} }
//
// Host imports (module "herald"):
//   - log(level u32, ptr u32, len u32)
//   - get_context(outPtr u32, outCap u32) u32  — writes CallContext JSON, returns byte length
//   - http_fetch(reqPtr u32, reqLen u32, outPtr u32, outCap u32) u32 — requires network: permission
package wasm
