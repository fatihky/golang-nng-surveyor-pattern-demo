package cmd

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/satori/go.uuid"
	e3ch "github.com/soyking/e3ch"
	"github.com/spf13/cobra"
	// "github.com/theodesp/find-port"
	"go.etcd.io/etcd/clientv3"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/surveyor"
	_ "go.nanomsg.org/mangos/v3/transport/all"

	"github.com/paper2code/golang-nng-surveyor-pattern-demo/pkg/models"
)

var (
	serverAddress    string
	surveyorAddress  string
	surveyorTimeout  int
	surveyorMessages cmap.ConcurrentMap // Just for testing the aggregation of survey's responses and not having race conditions on standard maps
	e3Disabled       bool
	e3Endpoints      []string
	e3Client         *clientv3.Client
	e3chClient       *e3ch.EtcdHRCHYClient
)

// surveyor thread
type survey struct {
	query []byte
	ch    chan surveyresult
}

type surveyresult struct {
	err error
	msg []byte
}

func surveyorReceiverThread(sock mangos.Socket) chan surveyresult {
	ch := make(chan surveyresult, 1000)

	go func() {
		for {
			msg, err := sock.Recv()
			result := surveyresult{
				err: err,
				msg: msg,
			}
			log.Println("surveyorReceiverThread: received a message")
			if err != nil {
				log.Printf("surveyorReceiverThread: recv error: %s\n", err)
			}
			ch <- result
		}
	}()

	return ch
}

func surveyorThread(surveys chan survey) {
	sock, err := newSurveyor(surveyorAddress)

	if err != nil {
		log.Warnf("newSurveyor.Error: %s", err)
		return
	}

	resultsch := surveyorReceiverThread(sock)

	defer sock.Close()

	queue := make([]survey, 10)

	for {
		select {
		case surv := <-surveys:
			if err = sock.Send(surv.query); err != nil {
				log.Warnf("Failed sending survey: %s", err)
				// todo: return 400 message
				return
			}

			queue = append(queue, surv)
		case result := <-resultsch:
			surv := queue[0]
			queue = queue[1:]
			surv.ch <- result
		case <-time.After(30 * time.Second):
			fmt.Println("No requests received for 30 seconds")
		}
	}
}

// server
var ServerCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"s"},
	Short:   "Start the surveyor server",
	Long:    "Start the surveyor restful server",
	Run: func(cmd *cobra.Command, args []string) {
		surveys := make(chan survey, 1000)

		go surveyorThread(surveys)

		// init etcd kv store
		// initial etcd v3 client
		var err error
		if !e3Disabled {
			e3Client, err = clientv3.New(clientv3.Config{Endpoints: e3Endpoints})
			if err != nil {
				log.Fatal(err)
			}

			// new e3ch client with namespace(rootKey)
			e3chClient, err = e3ch.New(e3Client, "surveyor")
			if err != nil {
				log.Fatal(err)
			}

			// set the rootKey as directory
			err = e3chClient.FormatRootKey()
			if err != nil {
				log.Fatal(err)
			}
		}

		// init default gin-gonic instance
		r := gin.Default()

		// setup a basic GET endpoint for test
		r.GET("/surveyor/:query", func(c *gin.Context) {
			start := time.Now()
			query := c.Params.ByName("query")

			// openPort, err := findport.DetectOpenPort()
			// if err != nil {
			// 	log.Fatalf("Get available port failed with %v", err)
			// }

			// log.Infof("Found available port at: %v\n", openPort)

			// todo. assert that query is not empty, either return 400 message

			time.Sleep(time.Second / 8)

			reponse := make(map[string]interface{}, 0)
			results := make(map[string]interface{}, 0)
			for {
				// Prepare query to respondent
				respondentQuery := &models.Query{
					CreatedAt: time.Now(),
					Query:     query,
					UUID:      uuid.NewV4().String(),
				}
				respondentQueryBytes, err := json.Marshal(&respondentQuery)
				if err != nil {
					log.Fatal(err)
				}

				reponse["query"] = respondentQuery
				// sending the auery to the respondent
				log.Info("SERVER: SENDING DATE SURVEY REQUEST")

				// [START - OLD SURVEY IMPLEMENTATION]
				// if err = sock.Send(respondentQueryBytes); err != nil {
				// 	log.Warnf("Failed sending survey: %s", err)
				// 	// todo: return 400 message
				// 	return
				// }

				// waiting for replies from respondent
				// todo: return aggregated results with 200 status
				// var msg []byte
				// for {
				// 	if msg, err = sock.Recv(); err != nil {
				// 		// log.Warnf("Failed receiving survey response: %s", err)
				// 		// todo: return 400 message
				// 		break
				// 	} else {
				// 		// unserialize response
				// 		var respondentResponse models.Response
				// 		if err := json.Unmarshal(msg, &respondentResponse); err != nil {
				// 			log.Fatal(err)
				// 		}
				// 		results[respondentResponse.ServiceName] = respondentResponse
				// 		log.Infof("SERVER: RECEIVED \"%s\" SURVEY RESPONSE\n", string(msg))
				// 	}
				// }
				// [END - OLD SURVEY IMPLEMENTATION]

				// [START - NEW SURVEY IMPLEMENTATION]
				resultch := make(chan surveyresult)
				surv := survey{
					query: respondentQueryBytes,
					ch:    resultch,
				}
				surveys <- surv
				result := <-resultch

				if result.err != nil {
					log.Warnf("Survey failed: %s", result.err)
				} else {
					// unserialize response
					var respondentResponse models.Response
					if err := json.Unmarshal(result.msg, &respondentResponse); err != nil {
						log.Fatal(err)
					}
					results[respondentResponse.ServiceName] = respondentResponse
					log.Infof("SERVER: RECEIVED \"%s\" SURVEY RESPONSE\n", string(result.msg))
				}
				// [END - NEW SURVEY IMPLEMENTATION]

				reponse["results"] = results
				end := time.Now()
				responseTime := end.Sub(start)
				timerMs := int64(responseTime / time.Millisecond)
				timerNs := int64(responseTime / time.Nanosecond)
				reponse["timer_ms"] = timerMs
				log.Infof("SERVER: SURVEY OVER, took %d ms, %d ns", timerMs, timerNs)
				// sock.Close()
				c.IndentedJSON(http.StatusOK, reponse)
				break
			}
			return
		})

		if err := r.Run(serverAddress); err != nil {
			log.Fatalf("Error: %v", err)
		}

	},
}

func init() {
	rand.Seed(time.Now().Unix())
	ServerCmd.Flags().BoolVarP(&e3Disabled, "no-etcd", "", false, "Disable etcd support.")
	ServerCmd.Flags().StringSliceVarP(&e3Endpoints, "etcd-endpoints", "", []string{"127.0.0.1:2379"}, "Etcd server address")
	ServerCmd.Flags().StringVarP(&serverAddress, "server-address", "", "0.0.0.0:3200", "HTTP server Address")
	ServerCmd.Flags().StringVarP(&surveyorAddress, "surveyor-address", "", "tcp://localhost:40999", "Surveyor Address")
	ServerCmd.Flags().IntVarP(&surveyorTimeout, "surveyor-timeout", "", 20000, "Surveyor request timeout in millisecond")
	RootCmd.AddCommand(ServerCmd)
}

func newSurveyor(url string) (sock mangos.Socket, err error) {
	if sock, err = surveyor.NewSocket(); err != nil {
		log.Warnf("can't get new surveyor socket: %s", err)
		return nil, err
	}
	if err = sock.Listen(url); err != nil {
		log.Warnf("can't listen on surveyor socket: %s", err)
		return nil, err
	}
	err = sock.SetOption(mangos.OptionSurveyTime, time.Second/2)
	if err != nil {
		log.Warnf("SetOption(): %s", err)
		return nil, err
	}
	return
}
