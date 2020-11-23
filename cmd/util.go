package cmd

import (
	"fmt"
	"os"
	"time"

	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/pub"
	"go.nanomsg.org/mangos/v3/protocol/sub"
)

func die(format string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func createSockRecvChannel(sock mangos.Socket, tmo time.Duration, cls chan bool) chan []byte {
	messages := make(chan []byte)

	// setup socket recive timeout
	sock.SetOption(mangos.OptionRecvDeadline, tmo)

	// receive messages
	go func() {
		for {
			exit := false

			select {
			case <-cls:
				exit = true
				break
			default:
				msg, err := sock.Recv()

				if err != nil {
					if err != mangos.ErrRecvTimeout {
						log.Fatalf("Failed to receive a message from socket: %s. socket name: %s\n", err, sock.Info().SelfName)
					}
				} else {
					messages <- msg
				}
			}

			if exit {
				break
			}
		}
	}()

	return messages
}

func createPubsubSocket(endpoint string, publish bool, bind bool) (mangos.Socket, error) {
	var sock mangos.Socket
	var err error

	if publish {
		if sock, err = pub.NewSocket(); err != nil {
			return nil, err
		}
	} else {
		if sock, err = sub.NewSocket(); err != nil {
			return nil, err
		}

		// subscribe to all messages
		err = sock.SetOption(mangos.OptionSubscribe, []byte(""))

		if err != nil {
			return nil, err
		}
	}

	if bind {
		if err = sock.Listen(endpoint); err != nil {
			return nil, err
		}
	} else {
		if err = sock.Dial(endpoint); err != nil {
			return nil, err
		}
	}

	return sock, nil
}
