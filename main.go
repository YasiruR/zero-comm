package main

import (
	"github.com/YasiruR/didcomm-prober/cli"
)

// init prober with args recipient name
// output public key

// set recipient{name, endpoint, public key}

// send message

func main() {
	cli.Init(`test`, `keyyy`, `localhost`)
	cli.BasicCommands()
}
