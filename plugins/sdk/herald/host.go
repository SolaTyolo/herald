//go:build wasip1

// Package herald provides TinyGo helpers for Herald WASM provider host imports.
package herald

//go:wasmimport herald log
func hostLog(level, ptr, len uint32)

//go:wasmimport herald get_context
func hostGetContext(outPtr, outCap uint32) uint32

//go:wasmimport herald http_fetch
func hostHTTPFetch(reqPtr, reqLen, outPtr, outCap uint32) uint32

const (
	LogDebug = 0
	LogInfo  = 1
	LogWarn  = 2
	LogError = 3
)
