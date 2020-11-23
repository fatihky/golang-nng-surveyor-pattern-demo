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

	surveyorConnectEndpoint string
	querySubEndpoint        string
	queryResultPubEndpoint  string
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

		go cliMain(surveyorConnectEndpoint, querySubEndpoint, queryResultPubEndpoint, clsMain)

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
	ClientCmd.Flags().StringVarP(&surveyorConnectEndpoint, "surveyor-endpoint", "", "tcp://localhost:40700", "Surveyor endpoint")
	ClientCmd.Flags().StringVarP(&querySubEndpoint, "query-sub-endpoint", "", "tcp://localhost:40701", "Query subscribe endpoint")
	ClientCmd.Flags().StringVarP(&queryResultPubEndpoint, "result-pub-endpoint", "", "tcp://localhost:40702", "Query result publish endpoint")
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

// [START Survey Responder]
func runRespondent(sock mangos.Socket, cls chan bool) {
	for {
		exit := false

		// receive and reply survey messages
		_, err := sock.Recv()

		if err != nil {
			if err != mangos.ErrRecvTimeout {
				log.Fatalf("respondent: can't receive survey message: %s\n", err)
			}
		} else {
			if err = sock.Send([]byte("IamAlive")); err != nil {
				log.Fatalf("respondent sock: send error: %s", err)
			}
		}

		// check if we received close signal
		select {
		case exit = <-cls:
			break
		default:
			break
		}

		if exit {
			log.Print("Respondent: exit signal received. Exiting...")
			break
		}
	}

	log.Print("respondent: exit")
}

// [END Survey Responder]

// [START Query Processor]
func processQuery(pub mangos.Socket, query query) {
	log.Printf("process query: %s\n", query.Question)

	result := queryresult{
		err:   nil,
		ID:    query.ID,
		Items: []string{"test", "result"},
	}

	if jsonBytes, err := json.Marshal(&result); err != nil {
		log.Warnf("Can't encode query result: %s\n", err)
	} else {
		log.Printf("query result json: %s\n", string(jsonBytes))
		pub.Send(jsonBytes)
	}
}

// [END Query Processor]

// [START Client]
func cliMain(surveyorEndpoint string, querySubscribeEndpoint string, resultPublishEndpoint string, cls chan bool) {
	var pub mangos.Socket
	var sub mangos.Socket
	clsResp := make(chan bool)
	clsSub := make(chan bool)
	clsSockRecvChannel := make(chan bool)
	respondentSock, err := newRespondent(surveyorEndpoint)

	if err != nil {
		log.Fatalf("Could not create a new socket: %s\n", err)
	}

	sub, err = createPubsubSocket(querySubscribeEndpoint, false, false)

	if err != nil {
		die("failed to create query subscribe socket: %s", err)
	}

	pub, err = createPubsubSocket(resultPublishEndpoint, true, false)

	if err != nil {
		die("failed to create result publish socket: %s", err)
	}

	subch := createSockRecvChannel(sub, time.Second, clsSub)
	// respondentMessages := sockRecvChannel(respondentSock, 5*time.Second, clsSockRecvChannel)

	go runRespondent(respondentSock, clsResp)

	for {
		exit := false

		select {
		case msg := <-subch:
			{
				var query query

				if err := json.Unmarshal(msg, &query); err != nil {
					log.Warnf("Can't decode query json: %s\n", err)

					break
				}

				go processQuery(pub, query)
			}
		case <-cls:
			clsSockRecvChannel <- true
			clsSub <- true
			exit = true
		}

		if exit {
			break
		}
	}
}

// [END Client]
