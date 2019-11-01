package main

import (
	"k8s-proxy/cmd"
	"os"
)

func main() {
	os.Exit(cmd.NewProxyCommand().Run())
}
