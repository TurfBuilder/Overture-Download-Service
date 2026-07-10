package downloader

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// connectToStream connects to the NATS server and returns the connection and a
// JetStream context. The caller is responsible for closing the connection.
func connectToStream(host string, port int) (*nats.Conn, jetstream.JetStream, error) {
	nc, err := nats.Connect(fmt.Sprintf("nats://%s:%d", host, port))
	if err != nil {
		return nil, nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, err
	}

	return nc, js, nil
}
