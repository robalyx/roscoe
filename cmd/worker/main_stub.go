//go:build !js || !wasm

package main

func main() {
	// This file exists to satisfy Go tooling when building for non-wasm targets
	panic("This program can only be built for WebAssembly")
}
