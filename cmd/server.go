package cmd

import (
	"fmt"
	"time"

	// "github.com/k0kubun/pp"
        "github.com/spf13/cobra"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/surveyor"

	// register transports
	_ "go.nanomsg.org/mangos/v3/transport/all"
)

var (
  serverURL string
)

var ServerCmd = &cobra.Command{
	Use:     "server <url>",
	Aliases: []string{"s"},
	Short:   "Start the server",
	Long:    "Start the nng server",
	Run: func(cmd *cobra.Command, args []string) {
	   server(serverURL)
	},
}

func init() {
	ServerCmd.Flags().StringVarP(&serverURL, "url", "", "tcp://search:40899", "Server URL")
	RootCmd.AddCommand(ServerCmd)
}

func server(url string) {
	var sock mangos.Socket
	var err error
	var msg []byte
	if sock, err = surveyor.NewSocket(); err != nil {
		die("can't get new surveyor socket: %s", err)
	}
	if err = sock.Listen(url); err != nil {
		die("can't listen on surveyor socket: %s", err.Error())
	}
	err = sock.SetOption(mangos.OptionSurveyTime, time.Second/2)
	if err != nil {
		die("SetOption(): %s", err.Error())
	}
	for {
		time.Sleep(time.Second)
		fmt.Println("SERVER: SENDING DATE SURVEY REQUEST")
		if err = sock.Send([]byte("DATE")); err != nil {
			die("Failed sending survey: %s", err.Error())
		}
		for {
			if msg, err = sock.Recv(); err != nil {
				break
			}
			fmt.Printf("SERVER: RECEIVED \"%s\" SURVEY RESPONSE\n",	string(msg))
		}
		fmt.Println("SERVER: SURVEY OVER")
	}
}
