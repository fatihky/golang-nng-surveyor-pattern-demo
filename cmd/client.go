package cmd

import (
	"time"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/spf13/cobra"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/respondent"
	_ "go.nanomsg.org/mangos/v3/transport/all"

	"github.com/paper2code/golang-nng-surveyor-pattern-demo/pkg/models"
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
		clsMain := make(chan bool)

		// give some time to server to spin up
		time.Sleep(time.Second / 2)

		go cliMain(clsMain)

		for {
			// sock, err := newRespondent(respondentSurveyor)
			sock, err := newRespondent("tcp://0.0.0.0:7964")
			if err != nil {
				// log.Fatal(err)
				continue
			}
			var msg []byte
			for {
				if msg, err = sock.Recv(); err != nil {
					log.Warnf("Cannot recv: %s", err)
				} else {
					// parse surveyor request
					var surveyorQuery models.Query
					if err := json.Unmarshal(msg, &surveyorQuery); err != nil {
						log.Fatal(err)
					}

					gofakeit.Seed(0)

					// generate an answer to the surveyor request
					log.Infof("CLIENT(%s): RECEIVED \"%s\" SURVEY REQUEST\n", respondentName, surveyorQuery.Query)
					var results []string
					results = append(results, gofakeit.HackerPhrase())
					now := time.Now()
					respondentAnswer := &models.Response{
						CreatedAt:   now,
						ServiceName: respondentName,
						Query:       surveyorQuery.Query,
						Results:     interfaceSlice(results),
					}

					// calculate the response time
					responseTime := now.Sub(surveyorQuery.CreatedAt)
					respondentAnswer.ResponseTimeMs = int64(responseTime / time.Millisecond)
					respondentAnswer.ResponseTimeNs = int64(responseTime / time.Nanosecond)
					respondentAnswerBytes, err := json.Marshal(&respondentAnswer)
					if err != nil {
						log.Fatal(err)
					}

					log.Infof("CLIENT(%s): SENDING DATE SURVEY RESPONSE\n", respondentName)

					// send response to the surveyor
					if err = sock.Send(respondentAnswerBytes); err != nil {
						log.Fatalf("Cannot send: %s", err)
					}
				}
			}
			sock.Close()
		}
	},
}

func init() {
	ClientCmd.Flags().StringVarP(&respondentSurveyor, "respondent-surveyor", "", "tcp://localhost:40899", "Respondent's surveyor address")
	ClientCmd.Flags().StringVarP(&respondentName, "respondent-name", "", "client0", "Client Name")
	RootCmd.AddCommand(ClientCmd)
}

func newRespondent(url string) (sock mangos.Socket, err error) {
	if sock, err = respondent.NewSocket(); err != nil {
		log.Warnf("can't get new respondent socket: %s", err)
		return nil, err
	}
	if err = sock.Dial(url); err != nil {
		//log.Warnf("can't dial on respondent socket: %s", err)
		return nil, err
	}
	return
}

func cliMain(cls chan bool) {
	clsSockRecvChannel := make(chan bool)
	respondentSock, err := newRespondent(respondentSurveyor)

	if err != nil {
		log.Fatalf("Could not create a new socket: %s\n", err)
	}

	// respondentMessages := sockRecvChannel(respondentSock, 5*time.Second, clsSockRecvChannel)

	for {
		exit := false

		select {
		case <-time.After(time.Second / 8):
			{
				_, err := respondentSock.Recv()

				if err != nil {
					if err == mangos.ErrRecvTimeout {
						break
					}

					log.Fatalf("respondent: can't receive survey message: %s\n", err)
				}

				if err = respondentSock.Send([]byte("IamAlive")); err != nil {
					log.Fatalf("respondent sock: send error: %s", err)
				}
			}
		case <-cls:
			clsSockRecvChannel <- true
			exit = true
		}

		if exit {
			break
		}
	}
}
