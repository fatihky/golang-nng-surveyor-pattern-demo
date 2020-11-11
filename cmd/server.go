package cmd

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/satori/go.uuid"
	e3ch "github.com/soyking/e3ch"
	"github.com/spf13/cobra"
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
	e3Endpoints      []string
	e3Client         *clientv3.Client
	e3chClient       *e3ch.EtcdHRCHYClient
)

var ServerCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"s"},
	Short:   "Start the surveyor server",
	Long:    "Start the surveyor restful server",
	Run: func(cmd *cobra.Command, args []string) {

		// init etcd kv store
		// initial etcd v3 client
		var err error
		e3Client, err = clientv3.New(clientv3.Config{Endpoints: []string{"127.0.0.1:2379"}})
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

		// init default gin-gonic instance
		r := gin.Default()

		// setup a basic GET endpoint for test
		r.GET("/surveyor/:query", func(c *gin.Context) {
			start := time.Now()
			query := c.Params.ByName("query")
			// todo. assert that query is not empty, either return 400 message
			sock, err := newSurveyor(surveyorAddress)
			if err != nil {
				log.Warnf("newSurveyor.Error: %s", err)
				// todo: return 400 message
				return
			}
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
				fmt.Println("SERVER: SENDING DATE SURVEY REQUEST")
				if err = sock.Send(respondentQueryBytes); err != nil {
					log.Warnf("Failed sending survey: %s", err)
					// todo: return 400 message
					return
				}

				// waiting for replies from respondent
				// todo: return aggregated results with 200 status
				var msg []byte
				for {
					if msg, err = sock.Recv(); err != nil {
						// log.Warnf("Failed receiving survey response: %s", err)
						// todo: return 400 message
						break
					} else {
						// unserialize response
						var respondentResponse models.Response
						if err := json.Unmarshal(msg, &respondentResponse); err != nil {
							log.Fatal(err)
						}
						results[respondentResponse.ServiceName] = respondentResponse
						log.Infof("SERVER: RECEIVED \"%s\" SURVEY RESPONSE\n", string(msg))
					}
				}
				reponse["results"] = results
				end := time.Now()
				responseTime := end.Sub(start)
				timerMs := int64(responseTime / time.Millisecond)
				timerNs := int64(responseTime / time.Nanosecond)
				reponse["timer_ms"] = timerMs
				log.Infof("SERVER: SURVEY OVER, took %d ms, %d ns", timerMs, timerNs)
				sock.Close()
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
