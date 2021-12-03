package main

import (
	"log"
	"net/http"

	godddns "github.com/tobychui/go-DDDNS/godddns"
)

//Demo function for validate user account
func ValidateCred(username string, password string) bool {
	return true
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
		log.Println("Client Router Started")
		http.ListenAndServe(":8082", clientHandler)
	}()

	//Add server node into the client list
	clientRouter.AddNode("server", "http://127.0.0.1:8081/connect", "")

	//Add client node into the server list
	serverRouter.AddNode("client", "http://127.0.0.1:8082/connect", "")

	//Generate client -> server TOTP
	clientToServer, err := clientRouter.StartConnection("server", "user", "123456")
	if err != nil {
		log.Fatal(err)
	}

	//Generate server -> client TOTP
	serverToClient, err := serverRouter.StartConnection("client", "user", "123456")
	if err != nil {
		log.Fatal(err)
	}

	log.Println(clientToServer, serverToClient)

	//Do a blocking lop
	select {}
}
