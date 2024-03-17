package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

type TunnelClient struct {
	listenAddress string
	dialer        net.Conn
	channel       chan struct{}
}

func NewTunnelClient(address string) *TunnelClient {
	return &TunnelClient{
		listenAddress: address,
		channel:       make(chan struct{}),
	}
}

func (s *TunnelClient) ListenForTunnel() error {
	dialer, err := net.Dial("tcp", s.listenAddress)
	if err != nil {
		return err
	}
	defer dialer.Close()
	s.dialer = dialer

	var connectionUrl string
	fmt.Print("Enter a local url string : ")
	fmt.Scanln(&connectionUrl)

	go s.writeToServer(true, "r.singharia,"+connectionUrl)
	go s.readHttpRequestFromServer()
	<-s.channel
	return nil
}

func (s *TunnelClient) writeToServer(isConnection bool, message string) error {
	var stringMessage []byte
	if isConnection {
		stringMessage = []byte("connection," + message)
	} else {
		stringMessage = []byte("data," + message)
	}
	_, err := s.dialer.Write(stringMessage)
	if err != nil {
		fmt.Println("Error writing dialer:", err)
		return err
	}
	fmt.Println("Message sent to server:", string(stringMessage))
	return nil
}

func (s *TunnelClient) readHttpRequestFromServer() error {
	buff := make([]byte, 2048)
	for {
		n, err := s.dialer.Read(buff)
		if err != nil {
			fmt.Println("Error reading from client: ", err)
			break
		}
		fmt.Println(string(buff[:n]))
		tcpResponse := string(buff[:n])
		splitTcpResponse := strings.Split(tcpResponse, ",")

		if splitTcpResponse[0] == "port" {
			fmt.Println("Your HTTP server is active on : ", splitTcpResponse[1])
			continue
		}

		proxyReq, err := http.NewRequest(splitTcpResponse[0], splitTcpResponse[1], strings.NewReader(splitTcpResponse[2]))

		if err != nil {
			fmt.Println("Error in creating request object: ", err)
			continue
		}

		httpClient := http.Client{}
		proxyResponse, err := httpClient.Do(proxyReq)
		if err != nil {
			fmt.Println("Error in proxy request :", splitTcpResponse, err)
			continue
		}
		defer proxyResponse.Body.Close()

		if proxyResponse.StatusCode == http.StatusOK {
			bodyBytes, err := io.ReadAll(proxyResponse.Body)
			if err != nil {
				log.Fatal(err)
			}
			bodyString := string(bodyBytes)
			fmt.Println("Response from the proxy request : ", bodyString)
			go s.writeToServer(false, bodyString)
		}
	}
	return nil
}

func main() {

	url := "http://localhost:8080/create-tunnel"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	fmt.Println("Response from server:", string(body))
	tunnelClient := NewTunnelClient(":" + string(body))
	log.Fatal(tunnelClient.ListenForTunnel())
}
