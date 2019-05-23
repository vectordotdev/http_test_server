package main

import (
	"log"
	"os"

	"gopkg.in/urfave/cli.v1"
)

func main() {
	app := cli.NewApp()
	app.Name = "http_test_server"
	app.Usage = "Simple HTTP server that is useful for testing purposes."
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "file, f",
			Usage: "A file to write messages to",
		},
	}
	app.Action = func(ctx *cli.Context) error {
		file := ctx.String("file")
		server := NewServer(addr, file)
		server.Listen()
		defer server.Close()

		if server.File != nil {
			log.Println("Closing file")
			server.File.Close()
		}

		log.Printf("Received %v requests", server.MessageCount, server.ConnectionCount)

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
