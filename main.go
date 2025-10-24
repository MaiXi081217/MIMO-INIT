package main

import (
	"flag"
	"fmt"
	"resourcemgr/internal/run"
)

func main() {
	help   := flag.Bool("help", false, "Show help message")
	update := flag.Bool("sys", false, "System update mode")
	tgt    := flag.Bool("target", false, "Target update mode")

	flag.Parse()

	if *help {
		fmt.Println(`mimo-update: One-step MIMO system resource updater
Usage:
	mimo-update [options]
Options:
	--help   Show help message
	--sys    Execute system update
	--target Execute target update`)
		return
	}

	if *update {
		run.RunUpdate()
		return
	}
	if *tgt {
		run.RuntgtUpdate()
		return
	}

	fmt.Println("Specify one of the options: -help")
}
