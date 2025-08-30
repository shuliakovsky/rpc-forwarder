package main

import "fmt"

var (
	Version    = "0.0.1"
	CommitHash = ""
)

func PrintVersion() {
	fmt.Printf("rpc-forwarder version: %s\n", Version)
	if CommitHash != "" {
		fmt.Printf("commit hash: %s\n", CommitHash)
	}
}
