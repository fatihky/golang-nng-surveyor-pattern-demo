package cmd

import (
	"fmt"

        "github.com/spf13/cobra"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/respondent"
	// register transports
	_ "go.nanomsg.org/mangos/v3/transport/all"
)

var (
	clientName string
	clientURL string
)

var ClientCmd = &cobra.Command{
        Use:     "client <url>",
        Aliases: []string{"c"},
        Short:   "Start the client",
        Long:    "Start the nng client",
        Run: func(cmd *cobra.Command, args []string) {
           client(serverURL, clientName)
        },
}

func init() {
        ClientCmd.Flags().StringVarP(&clientURL, "surveyor-url", "", serverURL, "Suerveryor URL")
        ClientCmd.Flags().StringVarP(&clientName, "name", "", "client0", "Client Name")
        RootCmd.AddCommand(ClientCmd)
}

func client(url string, name string) {
	var sock mangos.Socket
	var err error
	var msg []byte

	if sock, err = respondent.NewSocket(); err != nil {
		die("can't get new respondent socket: %s", err.Error())
	}
	if err = sock.Dial(url); err != nil {
		die("can't dial on respondent socket: %s", err.Error())
	}
	for {
		if msg, err = sock.Recv(); err != nil {
			die("Cannot recv: %s", err.Error())
		}
		fmt.Printf("CLIENT(%s): RECEIVED \"%s\" SURVEY REQUEST\n",
			name, string(msg))

		d := date()
		fmt.Printf("CLIENT(%s): SENDING DATE SURVEY RESPONSE\n", name)
		if err = sock.Send([]byte(d)); err != nil {
			die("Cannot send: %s", err.Error())
		}
	}
}
