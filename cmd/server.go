package cmd

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/spf13/cobra"
	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/surveyor"
	_ "go.nanomsg.org/mangos/v3/transport/all"
)

var (
	serverAddress    string
	surveyorAddress  string
	surveyorMessages cmap.ConcurrentMap // Just for testing the aggregation of survey's responses and not having race conditions on standard maps
)

var ServerCmd = &cobra.Command{
	Use:     "server",
	Aliases: []string{"s"},
	Short:   "Start the surveyor server",
	Long:    "Start the surveyor restful server",
	Run: func(cmd *cobra.Command, args []string) {

		// init default gin-gonic instance
		r := gin.Default()

		// setup a basic GET endpoint for test
		r.GET("/surveyor/:query", func(c *gin.Context) {
			query := c.Params.ByName("query")
			// todo. assert that query is not empty, either return 400 message
			sock, err := newSurveyor(surveyorAddress)
			if err != nil {
				// todo: return 400 message
				return
			}
			for {
				fmt.Println("SERVER: SENDING DATE SURVEY REQUEST")
				if err = sock.Send([]byte(query)); err != nil {
					log.Warnln("Failed sending survey: %s", err)
					// todo: return 400 message
					return
				}
				var msg []byte
				for {
					if msg, err = sock.Recv(); err != nil {
						log.Warnln("Failed receiving survey response: %s", err)
						// todo: return 400 message
						break
					}
					log.Infoln("SERVER: RECEIVED \"%s\" SURVEY RESPONSE\n", string(msg))

				}
				log.Info("SERVER: SURVEY OVER")
				// todo: return aggregated results with 200 status
			}
			return
		})

		if err := r.Run(serverAddress); err != nil {
			log.Fatalln("Error: %v", err)
		}

	},
}

func init() {
	ServerCmd.Flags().StringVarP(&serverAddress, "server-address", "", "0.0.0.0:3200", "HTTP server Address")
	ServerCmd.Flags().StringVarP(&surveyorAddress, "surveyor-address", "", "tcp://search:40899", "Surveyor Address")
	RootCmd.AddCommand(ServerCmd)
}

func newSurveyor(url string) (sock mangos.Socket, err error) {
	if sock, err = surveyor.NewSocket(); err != nil {
		log.Warnln("can't get new surveyor socket: %s", err)
		return nil, err
	}
	if err = sock.Listen(url); err != nil {
		log.Warnln("can't listen on surveyor socket: %s", err)
		return nil, err
	}
	err = sock.SetOption(mangos.OptionSurveyTime, time.Second/2)
	if err != nil {
		log.Warnln("SetOption(): %s", err)
		return nil, err
	}
	return
}
