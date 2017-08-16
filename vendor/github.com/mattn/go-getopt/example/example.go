package main

import . "github.com/mattn/go-getopt"
import "os"

func main() {
	var c int

	OptErr = 0
	for {
		if c = Getopt("a:bh"); c == EOF {
			break
		}
		switch c {
		case 'a':
			println("a=", OptArg)
		case 'b':
			println("i:on")
		case 'h':
			println("usage: example [-a foo|-b|-h]")
			os.Exit(1)
		}
	}

	for n := OptInd; n < len(os.Args); n++ {
		println(os.Args[n])
	}
}
