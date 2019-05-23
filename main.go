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
			Name:  "port, p",
			Usage: "The port to bind to",
		},
		cli.StringFlag{
			Name:  "file, f",
			Usage: "A file to write messages to",
		},
	}
	app.Action = func(ctx *cli.Context) error {
		port := ctx.String("port")

		if port == "" {
			message := "The port argument is required: `http_test_server -p 8080`"
			// Exit with 65, EX_DATAERR, to indicate input data was incorrect
			return cli.NewExitError(message, 65)
		}

		file := ctx.String("file")
		server := NewServer(port, file)
		server.Listen()

		if server.File != nil {
			log.Println("Closing file")
			server.File.Close()
		}

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
