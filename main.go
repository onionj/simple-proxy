package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/bepass-org/proxy/pkg/mixed"
	"github.com/bepass-org/proxy/pkg/statute"
	"github.com/onionj/websocket-mux/muxr"
)

var version = "" // set it just in Makefile

var (
	authToken string
	bind      string

	server     bool
	publicKey  string
	privateKey string
	path       string

	client     bool
	serverAddr string

	muxrClient *muxr.Client
	bufferSize int
)

func main() {
	flag.StringVar(&bind, "bind", ":1080", "The address to bind the server to")
	flag.StringVar(&authToken, "token", "1234", "Auth token for improve security")
	flag.IntVar(&bufferSize, "buffer-size", 4096, "default is 4096 (4KB)")

	// server
	flag.BoolVar(&server, "server", false, "Start a server")
	flag.StringVar(&publicKey, "public-key", "", "TLS certificate public key.")
	flag.StringVar(&privateKey, "private-key", "", "TLS certificate private key.")
	flag.StringVar(&path, "path", "/", "Listener path")

	// Client
	flag.BoolVar(&client, "client", false, "Start a client")
	flag.StringVar(&serverAddr, "server-address", "", "If you run code as a client, you have to set the server address")

	flag.Parse()

	fmt.Println("version:", version)

	if client {

		muxrClient = muxr.NewClient(serverAddr)
		closerFunc, err := muxrClient.StartForever()
		if err != nil {
			fmt.Println("muxrClient err:", err)
			return
		}
		defer closerFunc()

		proxy := mixed.NewProxy(
			mixed.WithBindAddress(bind),
			mixed.WithUserHandler(clientHandler),
		)

		if err := proxy.ListenAndServe(); err != nil {
			panic(err)
		}
	} else if server {
		server := muxr.NewServer(bind)
		server.Handle(path, serverHandler)

		if publicKey != "" && privateKey != "" {
			if err := server.ListenAndServeTLS(publicKey, privateKey); err != nil {
				panic(err)
			}

		} else {
			if err := server.ListenAndServe(); err != nil {
				panic(err)
			}
		}

	} else {
		flag.Usage()
	}
}

func clientHandler(req *statute.ProxyRequest) error {
	fmt.Println("handling request to", req.Network, req.Destination)

	stream, err := muxrClient.Dial()
	if err != nil {
		fmt.Println("clientHandler, muxrClient.Dial() err:", err)
		return err
	}
	defer stream.Close()

	// Send auth token
	err = stream.Write([]byte(authToken))
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Send destination network
	err = stream.Write([]byte(req.Network))
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Send destination address
	err = stream.Write([]byte(req.Destination))
	if err != nil {
		fmt.Println(err)
		return err
	}

	wait := make(chan struct{}, 2)

	// Send incoming data from the local system to the proxy server.
	buf := make([]byte, bufferSize)
	go func() {
		for {
			n, err := req.Conn.Read(buf)
			if err != nil {
				break
			}

			err = stream.Write(buf[:n])
			if err != nil {
				break
			}
		}
		wait <- struct{}{}
	}()

	// Send incoming data from the proxy server to the local system.
	go func() {
		for {
			data, err := stream.Read()
			if err != nil {
				break
			}
			_, err = req.Conn.Write(data)
			if err != nil {
				break
			}
		}
		wait <- struct{}{}
	}()
	<-wait
	return nil
}

func serverHandler(stream *muxr.Stream) {

	token, err := stream.Read()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	if authToken != string(token) {
		fmt.Println("Invalid token:", string(token))
		_ = stream.ConnAdaptor.Conn.Close()
		return
	}

	networkBuffer, err := stream.Read()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	network := string(networkBuffer)

	destinationBuffer, err := stream.Read()
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	destination := string(destinationBuffer)

	// check for private networks
	host, _, err := net.SplitHostPort(destination)
	if err == nil {
		if host == "localhost" {
			fmt.Println("drop request to ", destination)
			return
		}
		ip := net.ParseIP(host)
		if ip != nil && (ip.IsPrivate() || ip.IsLoopback()) {
			fmt.Println("drop request to ", destination)
			return
		}
	}

	fmt.Println("handling request to", network, destination)

	// Connecting to destinations
	conn, err := net.Dial(network, destination)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer func(conn net.Conn) {
		_ = conn.Close()
	}(conn)

	wait := make(chan struct{}, 2)

	// Send incoming data from the destination to the client.
	buf := make([]byte, bufferSize)
	go func() {
		for {
			n, err := conn.Read(buf)
			if err != nil {
				break
			}

			err = stream.Write(buf[:n])
			if err != nil {
				break
			}
		}
		wait <- struct{}{}
	}()

	// Send incoming data from the client to the destination.
	go func() {
		for {
			data, err := stream.Read()
			if err != nil {
				break
			}
			_, err = conn.Write(data)
			if err != nil {
				break
			}
		}
		wait <- struct{}{}
	}()
	<-wait
}
