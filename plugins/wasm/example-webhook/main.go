package main

import "encoding/json"

// Build: GOOS=wasip1 GOARCH=wasm go build -o provider.wasm .

//go:wasmexport malloc
func malloc(size uint32) uint32 {
	return uint32(len(make([]byte, size)))
}

//go:wasmexport Send
func Send(inPtr, inLen uint32) (outPtr, outLen uint32) {
	body, _ := json.Marshal(map[string]string{"providerRef": "webhook-stub"})
	ptr := malloc(uint32(len(body)))
	return ptr, uint32(len(body))
}

func main() {}
