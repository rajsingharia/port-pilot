package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"strings"
	// "github.com/google/uuid"
)

// type User struct {
// 	id int16
// }

// type UrlToTunnelMapping struct {
// 	mapping map[string]*Tunnel
// }

type Tunnel struct {
	tunnelUrl  string
	httpServerUrl string
	ln            net.Listener
	channel       chan struct{}
	msgChannel    chan []byte
	peerMap       map[string]net.Conn // key: client address, value: connection with the client
	httpServer    *http.Server        // server for http tunnel
}

func NewTunnel() *Tunnel {

	var tunnelPort int
	for {
		tunnelPort = rand.Intn(9000) + 1000
		if tunnelPort != 3000 && tunnelPort != 8080 {
			break
		}
	}

	var httpPort int
	for {
		httpPort = rand.Intn(9000) + 1000
		if httpPort != 3000 && httpPort != 8080 {
			break
		}
	}

	return &Tunnel{
		tunnelUrl: "http://localhost:" + fmt.Sprint(tunnelPort),
		httpServerUrl: "http://localhost:" + fmt.Sprint(httpPort),
		channel:       make(chan struct{}),
		msgChannel:    make(chan []byte, 10),
		peerMap:       make(map[string]net.Conn),
	}
}

func (s *Tunnel) StartTunnel() error {
	ln, err := net.Listen("tcp", s.tunnelUrl)
	if err != nil {
		return err
	}
	defer ln.Close()

	s.ln = ln

	go s.tunnelAcceptLoop()

	<-s.channel
	close(s.msgChannel)

	return nil
}

func (s *Tunnel) tunnelAcceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err)
			continue
		}
		fmt.Println("New connection to server:", conn.RemoteAddr())
		//Before adding  a new peer to the map, check if it is already there. If so, close the new one and continue
		if _, ok := s.peerMap[conn.RemoteAddr().String()]; ok {
			conn.Close()
			continue
		}
		s.peerMap[conn.RemoteAddr().String()] = conn
		// start listening for http request on port 1234 and send them using writeToTunnel

		go s.tunnelReadLoop(conn)

		go func() {
			for msg := range s.msgChannel {
				splitTcpResponse := strings.Split(string(msg), ",")
				if splitTcpResponse[0] == "connection" {
					fmt.Println("Message received for connection from client : ", splitTcpResponse[1]+", url = "+splitTcpResponse[2])
					go s.startHTTPServerListening(splitTcpResponse[2], conn)
					break
				}
			}
		}()
	}
}

func (s *Tunnel) startHTTPServerListening(url string, conn net.Conn) {

	handler := func(w http.ResponseWriter, r *http.Request) {
		// fmt.Printf("Request received for endpoint: %s\n", r.URL.Path)

		// // Write a response
		// fmt.Fprintf(w, "Hello, you've requested: %s\n", r.URL.Path)

		// Call writeToTunnel if defined
		requestStr := fmt.Sprintf("%s,%s,%s", r.Method, url, r.Body)
		writeToTunnel(conn, requestStr)

		for msg := range s.msgChannel {
			splitTcpResponse := strings.Split(string(msg), ",")
			if splitTcpResponse[0] == "data" {
				fmt.Println("Message received from client : ", string(msg))
				w.Write([]byte(msg))
				w.Header().Clone()
				break
			}
		}
	}

	go writeToTunnel(conn, "port," + s.httpServerUrl)

	fmt.Println("HTTP Server started listening on port : ", s.httpServerUrl)
	s.httpServer = &http.Server{
		Addr:    s.httpServerUrl,
		Handler: http.HandlerFunc(handler),
	}

	log.Fatal(s.httpServer.ListenAndServe())
}

func (s *Tunnel) tunnelReadLoop(conn net.Conn) {
	defer conn.Close()
	buff := make([]byte, 2048)
	for {
		n, err := conn.Read(buff)
		if err != nil {
			fmt.Println("Error reading from client: ", err)
			break
		}
		s.msgChannel <- buff[:n]
		//writeToTunnel(conn, "Thank you for your message!!")
	}
}

func writeToTunnel(conn net.Conn, message string) {
	conn.Write([]byte(message))
}

func listenForTunnelCreation() {

	handler := func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/create-tunnel") {
			tunnel := NewTunnel()
			log.Fatal(tunnel.StartTunnel())
			w.Write([]byte(tunnel.tunnelUrl))
		} else {
			w.Write([]byte("Invalid request"))
		}
	}

	server := http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(handler),
	}

	log.Fatal(server.ListenAndServe())
}

func main() {
	// tunnel := NewTunnel(":3000")
	// log.Fatal(tunnel.StartTunnel())
	listenForTunnelCreation()
}
