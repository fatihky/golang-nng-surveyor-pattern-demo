package cmd

import (
	"math/rand"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	cmap "github.com/orcaman/concurrent-map"
	uuid "github.com/satori/go.uuid"
	e3ch "github.com/soyking/e3ch"
	"github.com/spf13/cobra"
	"go.etcd.io/etcd/clientv3"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/surveyor"
	_ "go.nanomsg.org/mangos/v3/transport/all"
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

	queries                chan query
	surveyorEndpoint       string
	queryPubEndpoint       string
	queryResultSubEndpoint string
)

var ServerCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"s"},
	Short:   "Start the surveyor server",
	Long:    "Start the surveyor restful server",
	Run: func(cmd *cobra.Command, args []string) {
		closeMainCh := make(chan bool)

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

		// start main (surveyor, publisher and subscriber)
		go main(surveyorEndpoint, queryPubEndpoint, queryResultSubEndpoint, closeMainCh)

		// init default gin-gonic instance
		r := gin.Default()

		// setup a basic GET endpoint for test
		r.GET("/surveyor/:query", func(c *gin.Context) {
			response := make(map[string]interface{}, 0)
			start := time.Now()
			question := c.Params.ByName("query")
			query := query{
				ID:       uuid.NewV1().String(),
				Question: question,
				resultch: make(chan queryresult),
			}

			log.Println("question:", question) // delete me

			queries <- query
			result := <-query.resultch
			duration := time.Now().Sub(start)

			response["duration"] = duration

			if result.err != nil {
				response["success"] = false
				response["error"] = result.err.Error()
				response["results"] = []string{}
			} else {
				response["success"] = true
				response["error"] = nil
				response["results"] = result.Items
			}

			c.IndentedJSON(http.StatusOK, response)
		})

		// r.GET("/surveyorold/:query", func(c *gin.Context) {
		// 	start := time.Now()
		// 	query := c.Params.ByName("query")

		// 	openPort, err := findport.DetectOpenPort()
		// 	if err != nil {
		// 		log.Fatalf("Get available port failed with %v", err)
		// 	}

		// 	log.Infof("Found available port at: %v\n", openPort)

		// 	// todo. assert that query is not empty, either return 400 message
		// 	// sock, err := newSurveyor(surveyorAddress)
		// 	sock, err := newSurveyor("tcp://0.0.0.0:7964")
		// 	if err != nil {
		// 		log.Warnf("newSurveyor.Error: %s", err)
		// 		// todo: return 400 message
		// 		return
		// 	}
		// 	time.Sleep(time.Second / 8)

		// 	response := make(map[string]interface{}, 0)
		// 	results := make(map[string]interface{}, 0)
		// 	for {
		// 		// Prepare query to respondent
		// 		respondentQuery := &models.Query{
		// 			CreatedAt: time.Now(),
		// 			Query:     query,
		// 			UUID:      uuid.NewV4().String(),
		// 		}
		// 		respondentQueryBytes, err := json.Marshal(&respondentQuery)
		// 		if err != nil {
		// 			log.Fatal(err)
		// 		}

		// 		response["query"] = respondentQuery
		// 		// sending the auery to the respondent
		// 		log.Info("SERVER: SENDING DATE SURVEY REQUEST")
		// 		if err = sock.Send(respondentQueryBytes); err != nil {
		// 			log.Warnf("Failed sending survey: %s", err)
		// 			// todo: return 400 message
		// 			return
		// 		}

		// 		// waiting for replies from respondent
		// 		// todo: return aggregated results with 200 status
		// 		var msg []byte
		// 		for {
		// 			if msg, err = sock.Recv(); err != nil {
		// 				// log.Warnf("Failed receiving survey response: %s", err)
		// 				// todo: return 400 message
		// 				break
		// 			} else {
		// 				// unserialize response
		// 				var respondentResponse models.Response
		// 				if err := json.Unmarshal(msg, &respondentResponse); err != nil {
		// 					log.Fatal(err)
		// 				}
		// 				results[respondentResponse.ServiceName] = respondentResponse
		// 				log.Infof("SERVER: RECEIVED \"%s\" SURVEY RESPONSE\n", string(msg))
		// 			}
		// 		}
		// 		response["results"] = results
		// 		end := time.Now()
		// 		responseTime := end.Sub(start)
		// 		timerMs := int64(responseTime / time.Millisecond)
		// 		timerNs := int64(responseTime / time.Nanosecond)
		// 		response["timer_ms"] = timerMs
		// 		log.Infof("SERVER: SURVEY OVER, took %d ms, %d ns", timerMs, timerNs)
		// 		sock.Close()
		// 		c.IndentedJSON(http.StatusOK, response)
		// 		break
		// 	}
		// 	return
		// })

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
	ServerCmd.Flags().StringVarP(&surveyorEndpoint, "surveyor-endpoint", "", "tcp://localhost:40700", "Surveyor endpoint")
	ServerCmd.Flags().StringVarP(&queryPubEndpoint, "query-pub-endpoint", "", "tcp://localhost:40701", "Query publish endpoint")
	ServerCmd.Flags().StringVarP(&queryResultSubEndpoint, "query-result-sub-endpoint", "", "tcp://localhost:40702", "Query result subscribe endpoint")
	ServerCmd.Flags().IntVarP(&surveyorTimeout, "surveyor-timeout", "", 20000, "Surveyor request timeout in millisecond")

	RootCmd.AddCommand(ServerCmd)

	queries = make(chan query)
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
	err = sock.SetOption(mangos.OptionSurveyTime, time.Second)
	if err != nil {
		log.Warnf("SetOption(): %s", err)
		return nil, err
	}
	return
}

// CONSTANTS
var connectedServiceCount int32 = 0

// [START - Surveyor]
// Get service count by making a survey on the surveyor port
func getConnectedServiceCount(sock mangos.Socket) {
	var err error
	var cnt int32 = 0

	if err = sock.Send([]byte("whoisalive")); err != nil {
		log.Fatalf("surveyor: failed sending survey: %s\n", err)
	}

	for {
		if _, err = sock.Recv(); err != nil {
			if err == mangos.ErrRecvTimeout || err == mangos.ErrProtoState {
				break
			}

			log.Fatalf("surveyor: failed to recv survey result: %s\n", err)
		}

		cnt++
	}

	log.Printf("Connected service count: %d. Previous value: %d\n", cnt, atomic.LoadInt32(&connectedServiceCount))

	atomic.StoreInt32(&connectedServiceCount, cnt)
}

func runSurveyor(url string, close chan bool) {
	sock, err := newSurveyor(url)

	if err != nil {
		log.Fatalf("Can not create a new surveyor socket: %s\n", err)
	}

	// startup

	// get connected service count at first
	time.Sleep(time.Second)
	getConnectedServiceCount(sock)

	for {
		exit := false

		select {
		case <-time.After(10 * time.Second):
			log.Print("get connected device count")
			getConnectedServiceCount(sock)
		case <-close:
			exit = true
			break
		}

		if exit {
			break
		}
	}

	// cleanup
	sock.Close()
}

// [END - Surveyor]

// [START - Query Manager]
func runQueryManager(queryPublishEndpoint string, resultSubscribeEndpoint string, cls chan bool) {
	var pub mangos.Socket
	var sub mangos.Socket
	var subch chan []byte
	var err error
	var clsSub = make(chan bool)

	activeQueries := make(map[string]query)
	queryMetas := make(map[string]querymeta)

	pub, err = createPubsubSocket(queryPublishEndpoint, true, true)

	if err != nil {
		die("failed to create query publish socket: %s", err)
	}

	sub, err = createPubsubSocket(resultSubscribeEndpoint, false, true)

	if err != nil {
		die("failed to create result subscribe socket: %s", err)
	}

	subch = createSockRecvChannel(sub, 30*time.Second, clsSub)

	for {
		exit := false

		select {
		case query := <-queries:
			{
				log.Print("query received") // delete me
				if jsonBytes, err := json.Marshal(&query); err != nil {
					log.Warnf("Can't encode json: %s\n", err)

					result := queryresult{
						err: err,
					}

					query.resultch <- result
				} else {
					log.Printf("publish query id: %s\n", query.ID)
					activeQueries[query.ID] = query
					queryMetas[query.ID] = querymeta{
						timeoutAt:       time.Now().Add(query.maxExecutionTime),
						receivedResults: make([]queryresult, 0, atomic.LoadInt32(&connectedServiceCount)),
					}
					pub.Send(jsonBytes)
				}
			}
		case msg := <-subch:
			{
				log.Printf("message received on query result ch: %s\n", string(msg))

				var result queryresult

				if err := json.Unmarshal(msg, &result); err != nil {
					log.Warnf("Can't unmarshal query result: %s", err)
					break
				}

				log.Print("query result id:", result.ID)

				query := activeQueries[result.ID]
				meta := queryMetas[result.ID]

				if query.ID == result.ID {
					log.Print("query.ID == result.ID")
				}

				if meta2, ok := queryMetas[query.ID]; ok {
					log.Print("meta2 exists", meta2)
				} else {
					log.Print("meta2 not found!")
				}

				log.Print("query meta timeout at:", meta.timeoutAt)
				log.Print("query meta received result cap:", cap(meta.receivedResults))

				log.Print("before:", len(meta.receivedResults))
				meta.receivedResults = append(meta.receivedResults, result)
				log.Print("after:", len(meta.receivedResults))

				log.Printf("query received %d results so far. we've been waiting for %d results total.", len(meta.receivedResults), atomic.LoadInt32(&connectedServiceCount))

				if int32(len(meta.receivedResults)) == atomic.LoadInt32(&connectedServiceCount) {
					result := queryresult{
						err:   nil,
						ID:    query.ID,
						Items: make([]string, 0),
					}

					for i := 0; i < len(meta.receivedResults); i++ {
						received := meta.receivedResults[i]
						result.Items = append(result.Items, received.Items...)
					}

					query.resultch <- result

					delete(activeQueries, result.ID)
					delete(queryMetas, result.ID)
				}
			}
		case <-time.After(10 * time.Second):
			{
				// log.Printf("No query or query results received in last 10 seconds.\n")
			}
		case <-cls:
			{
				clsSub <- true
				exit = true
			}
		}

		if exit {
			break
		}
	}

	pub.Close()
	sub.Close()
}

// [END - Query Manager]

// Main
func main(surveyorUrl string, queryPublishEndpoint string, resultSubscribeEndpoint string, close chan bool) {
	clsSurveyor := make(chan bool)
	clsQueryManager := make(chan bool)

	go runSurveyor(surveyorUrl, clsSurveyor)
	go runQueryManager(queryPublishEndpoint, resultSubscribeEndpoint, clsQueryManager)

	select {
	case <-close:
		// send close signals to other threads
		clsSurveyor <- true
		clsQueryManager <- true
	}

	// wait for some time to cleanup
	<-time.After(5 * time.Second)
}
