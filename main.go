package main

import (
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	godddns "github.com/tobychui/go-DDDNS/godddns"
)

/*
	Go-DDDNS Example (I might want to change this name later)

	For first time compile & start up, this demo will do the followings:

	1. Create two service router for go-DDDNS
	2. Cross register both router to each other
	3. Lets them heartbeat and keep in sync with the other's IP address
	4. Write the configuration to json file

	For 2nd time startup, this demo will do the followings:

	1. Load service router from json config file
	2. Inject auth function into the loaded service router
	3. Resume heartbeat connections
*/

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

	var serverRouter *godddns.ServiceRouter
	var clientRouter *godddns.ServiceRouter
	if fileExists("serverRouter.json") {
		//Create service router from previous record
		r, err := godddns.NewRouterFromJSONFile("serverRouter.json")
		if err != nil {
			log.Fatal(err)
		}
		r.InjectAuthFunction(ValidateCred)

		serverRouter = r
	} else {
		//Create the testing server router
		serverRouter = godddns.NewServiceRouter(godddns.RouterOptions{
			DeviceUUID:   "server",
			AuthFunction: ValidateCred,
			SyncInterval: 10,
		})

		//Add client node into the server list
		s2cNode := serverRouter.NewNode("client", 8082, "/connect", "/heartbeat")
		serverRouter.AddNode(s2cNode)
	}

	//Start server router connection handler
	go func() {
		serverHandler.HandleFunc("/connect", serverRouter.HandleConnectionEstablishResponse)
		serverHandler.HandleFunc("/heartbeat", serverRouter.HandleHeartBeatRequest)
		log.Println("Server Router Started")
		http.ListenAndServe(":8081", serverHandler)
	}()

	if fileExists("clientRouter.json") {
		//Create service router from previous record
		r, err := godddns.NewRouterFromJSONFile("clientRouter.json")
		if err != nil {
			log.Fatal(err)
		}
		r.InjectAuthFunction(ValidateCred)

		clientRouter = r
	} else {
		//Create the client router
		clientRouter = godddns.NewServiceRouter(godddns.RouterOptions{
			DeviceUUID:   "client",
			AuthFunction: ValidateCred,
			SyncInterval: 10,
		})

		//Add server node into the client list
		c2sNode := clientRouter.NewNode("server", 8081, "/connect", "/heartbeat")
		clientRouter.AddNode(c2sNode)

	}

	//Start client router connection handler
	go func() {
		clientHandler.HandleFunc("/connect", clientRouter.HandleConnectionEstablishResponse)
		clientHandler.HandleFunc("/heartbeat", clientRouter.HandleHeartBeatRequest)
		log.Println("Client Router Started")
		http.ListenAndServe(":8082", clientHandler)
	}()

	//Generate client -> server TOTP
	clientToServer, err := clientRouter.StartConnection("server", "127.0.0.1", false, "user", "123456")
	if err != nil {
		log.Println("Unable to get TOTP from serverRouter", clientToServer)
		log.Fatal(err)
	}

	time.Sleep(1 * time.Second)

	//Generate server -> client TOTP
	serverToClient, err := serverRouter.StartConnection("client", "127.0.0.1", false, "user", "123456")
	if err != nil {
		log.Println("Unable to get TOTP from clientRouter", serverToClient)
		log.Fatal(err)
	}

	log.Println("TOTP Exchange done:", clientToServer, serverToClient)

	clientRouter.StartHeartBeat()
	serverRouter.StartHeartBeat()

	go func() {
		time.Sleep(11 * time.Second)
		//Export client Router
		js, _ := clientRouter.ExportRouterToJSON()
		ioutil.WriteFile("clientRouter.json", []byte(js), 0777)

		//Export server router
		js, _ = serverRouter.ExportRouterToJSON()
		ioutil.WriteFile("serverRouter.json", []byte(js), 0777)
		log.Println("Shutting down")
		os.Exit(0)
	}()
	//Do a blocking loop
	select {}
}

// Utilities
func fileExists(filename string) bool {
	if _, err := os.Stat(filename); errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}
