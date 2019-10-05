package main

import (
	"os"

	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "http_test_server"
	app.Usage = "Simple HTTP server that is useful for testing purposes."
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "address, a",
			Usage: "The address to bind to",
		},
	}
	app.Action = func(ctx *cli.Context) error {
		address := ctx.String("address")

		if address == "" {
			message := "The address argument is required: `http_test_server -a 0.0.0.0:8080`"
			// Exit with 65, EX_DATAERR, to indicate input data was incorrect
			return cli.NewExitError(message, 65)
		}

		server := NewServer(address)
		server.Listen()

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		os.Exit(1)
	}
}
