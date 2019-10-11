package main

import (
	"os"
	"varnish-purger/cmd"
)

func main() {
	os.Exit(cmd.NewVarnishPurgerCommand().Run())
}
