package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/respondent"
	_ "go.nanomsg.org/mangos/v3/transport/all"
)

const (
	respondentMessageTemplate = `- Service Name: %s
- Query: %s
- ReceivedAt: %s`
)

var (
	respondentName     string
	respondentSurveyor string
)

var ClientCmd = &cobra.Command{
	Use:     "client",
	Aliases: []string{"c"},
	Short:   "Start the client",
	Long:    "Start the nng client",
	Run: func(cmd *cobra.Command, args []string) {
		sock, err := newRespondent(respondentSurveyor)
		if err != nil {
			log.Fatal(err)
		}
		var msg []byte
		for {
			if msg, err = sock.Recv(); err != nil {
				log.Fatalf("Cannot recv: %s", err)
			}
			log.Infof("CLIENT(%s): RECEIVED \"%s\" SURVEY REQUEST\n", respondentName, string(msg))
			// generate an answer to the surveyor reauest
			respondentAnswer := fmt.Sprintf(respondentMessageTemplate, string(msg), respondentName, time.Now().Format("2006-02-01 01:01:01"))
			log.Infof("CLIENT(%s): SENDING DATE SURVEY RESPONSE\n", respondentName)
			if err = sock.Send([]byte(respondentAnswer)); err != nil {
				log.Fatalf("Cannot send: %s", err)
			}
		}
	},
}

func init() {
	ClientCmd.Flags().StringVarP(&respondentSurveyor, "respondent-surveyor", "", "tcp://search:40899", "Respondent's surveyor address")
	ClientCmd.Flags().StringVarP(&respondentName, "respondent-name", "", "client0", "Client Name")
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
