package main

import (
	"log"
	"net/http"

	godddns "github.com/tobychui/go-DDDNS/godddns"
)

//Demo function for validate user account
func ValidateCred(username string, password string) bool {
	if username == "user" && password == "123456" {
		return true
	} else {
		return false
	}
}

func main() {
	//Create two mux for the demo server handler
	serverHandler := http.NewServeMux()
	clientHandler := http.NewServeMux()

	//Create the testing server router
	serverRouter := godddns.NewServiceRouter(godddns.RouterOptions{
		DeviceUUID:   "server",
		AuthFunction: ValidateCred,
		SyncInterval: 10,
	})

	//Start server router connection handler
	go func() {
		serverHandler.HandleFunc("/connect", serverRouter.HandleConnectionEstablishResponse)
		serverHandler.HandleFunc("/heartbeat", serverRouter.HandleHeartBeatRequest)
		log.Println("Server Router Started")
		http.ListenAndServe(":8081", serverHandler)
	}()

	//Create the client router
	clientRouter := godddns.NewServiceRouter(godddns.RouterOptions{
		DeviceUUID:   "client",
		AuthFunction: ValidateCred,
		SyncInterval: 10,
	})

	//Start server router connection handler
	go func() {
		clientHandler.HandleFunc("/connect", clientRouter.HandleConnectionEstablishResponse)
		clientHandler.HandleFunc("/heartbeat", clientRouter.HandleHeartBeatRequest)
		log.Println("Client Router Started")
		http.ListenAndServe(":8082", clientHandler)
	}()

	//Add server node into the client list
	c2sNode := clientRouter.NewNode("server", 8081, "/connect", "/heartbeat")
	clientRouter.AddNode(c2sNode)

	//Add client node into the server list
	s2cNode := serverRouter.NewNode("client", 8082, "/connect", "/heartbeat")
	serverRouter.AddNode(s2cNode)

	//Generate client -> server TOTP
	clientToServer, err := clientRouter.StartConnection("server", "127.0.0.1", false, "user", "123456")
	if err != nil {
		log.Println("Unable to get TOTP from serverRouter", clientToServer)
		log.Fatal(err)
	}

	//Generate server -> client TOTP
	serverToClient, err := serverRouter.StartConnection("client", "127.0.0.1", false, "user", "123456")
	if err != nil {
		log.Println("Unable to get TOTP from clientRouter", serverToClient)
		log.Fatal(err)
	}

	log.Println(clientToServer, serverToClient)

	clientRouter.StartHeartBeat()
	serverRouter.StartHeartBeat()

	//Do a blocking loop
	select {}
}
