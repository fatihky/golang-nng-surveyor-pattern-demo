package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/respondent"
	_ "go.nanomsg.org/mangos/v3/transport/all"
)

var (
	respondentName     string
	respondentSurveyor string
)

var ClientCmd = &cobra.Command{
	Use:     "client <url>",
	Aliases: []string{"c"},
	Short:   "Start the client",
	Long:    "Start the nng client",
	Run: func(cmd *cobra.Command, args []string) {
		sock, err := newRespondent(respondentSurveyor)
		var msg []byte
		for {
			if msg, err = sock.Recv(); err != nil {
				die("Cannot recv: %s", err.Error())
			}
			fmt.Printf("CLIENT(%s): RECEIVED \"%s\" SURVEY REQUEST\n", respondentName, string(msg))
			d := date()
			fmt.Printf("CLIENT(%s): SENDING DATE SURVEY RESPONSE\n", respondentName)
			if err = sock.Send([]byte(d)); err != nil {
				die("Cannot send: %s", err.Error())
			}
		}
	},
}

func init() {
	ClientCmd.Flags().StringVarP(&respondentSurveyor, "repondent-surveyor", "", "", "Respondent's surveyor address")
	ClientCmd.Flags().StringVarP(&respondentName, "name", "", "client0", "Client Name")
	RootCmd.AddCommand(ClientCmd)
}

func newRespondent(url string) (sock mangos.Socket, err error) {
	if sock, err = respondent.NewSocket(); err != nil {
		log.Warnln("can't get new respondent socket: %s", err.Error())
		return nil, err
	}
	if err = sock.Dial(url); err != nil {
		log.Warnln("can't dial on respondent socket: %s", err.Error())
		return nil, err
	}
	return
}
