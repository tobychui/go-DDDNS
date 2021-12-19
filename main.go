package main

import (
	"errors"
	"fmt"
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

var (
	enableClientHTTP bool = true
	enableServerHTTP bool = true
	enableStaticHTTP bool = true
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
	//Create three mux for the demo server handler
	serverHandler := http.NewServeMux()
	clientHandler := http.NewServeMux()
	staticHandler := http.NewServeMux()

	var serverRouter *godddns.ServiceRouter
	var clientRouter *godddns.ServiceRouter
	var staticRouter *godddns.ServiceRouter

	var syncInterval int64 = 6

	/*
		SETTING UP 3DNS ROUTERS
	*/
	//Create the static router
	staticRouter = godddns.NewServiceRouter(godddns.RouterOptions{
		DeviceUUID:   "static",
		AuthFunction: ValidateCred,
		SyncInterval: syncInterval,
	})

	//Create the testing server router
	serverRouter = godddns.NewServiceRouter(godddns.RouterOptions{
		DeviceUUID:   "server",
		AuthFunction: ValidateCred,
		SyncInterval: syncInterval,
	})

	//Create the client router
	clientRouter = godddns.NewServiceRouter(godddns.RouterOptions{
		DeviceUUID:   "client",
		AuthFunction: ValidateCred,
		SyncInterval: syncInterval,
	})

	/*
		SETTING UP SERVER NODE LIST
	*/
	s2cNode := serverRouter.NewNode(godddns.NodeOptions{
		NodeID:        "client",
		Port:          8082,
		RESTInterface: "/godddns",
		RequireHTTPS:  false,
	})
	s2staticNode := serverRouter.NewNode(godddns.NodeOptions{
		NodeID:        "static",
		Port:          8083,
		RESTInterface: "/godddns",
		RequireHTTPS:  false,
	})
	serverRouter.AddNode(s2cNode)
	serverRouter.AddNode(s2staticNode)

	/*
		SETTING UP CLIENT NODE LIST
	*/
	c2sNode := clientRouter.NewNode(godddns.NodeOptions{
		NodeID:        "server",
		Port:          8081,
		RESTInterface: "/godddns",
		RequireHTTPS:  false,
	})
	c2staticNode := clientRouter.NewNode(godddns.NodeOptions{
		NodeID:        "static",
		Port:          8083,
		RESTInterface: "/godddns",
		RequireHTTPS:  false,
	})
	clientRouter.AddNode(c2sNode)
	clientRouter.AddNode(c2staticNode)

	/*
		SETTING UP STATIC NODE LIST
	*/
	static2sNode := staticRouter.NewNode(godddns.NodeOptions{
		NodeID:        "server",
		Port:          8081,
		RESTInterface: "/godddns",
		RequireHTTPS:  false,
	})
	static2cNode := staticRouter.NewNode(godddns.NodeOptions{
		NodeID:        "client",
		Port:          8082,
		RESTInterface: "/godddns",
		RequireHTTPS:  false,
	})
	staticRouter.AddNode(static2sNode)
	staticRouter.AddNode(static2cNode)

	/*
		CREATE ROUTER CONNECTION LISTENERS
	*/
	//Start server router connection handler
	go func() {
		serverHandler.HandleFunc("/godddns", func(w http.ResponseWriter, r *http.Request) {
			if enableServerHTTP {
				serverRouter.HandleConnections(w, r)
			} else {
				time.Sleep(10 * time.Second)
			}
		})
		log.Println("Server Router Started")
		http.ListenAndServe(":8081", serverHandler)
	}()

	//Start client router connection handler
	go func() {
		clientHandler.HandleFunc("/godddns", func(w http.ResponseWriter, r *http.Request) {
			if enableClientHTTP {
				clientRouter.HandleConnections(w, r)
			} else {
				time.Sleep(10 * time.Second)
			}
		})
		log.Println("Client Router Started")
		http.ListenAndServe(":8082", clientHandler)
	}()

	//Start static node connection handler
	go func() {
		staticHandler.HandleFunc("/godddns", func(w http.ResponseWriter, r *http.Request) {
			if enableStaticHTTP {
				staticRouter.HandleConnections(w, r)
			} else {
				time.Sleep(10 * time.Second)
			}
		})
		log.Println("Static Router Started")
		http.ListenAndServe(":8083", staticHandler)
	}()

	/*
		START CONNECTION
		There should be 6 connections

	*/

	time.Sleep(1 * time.Second)

	//Client to Server
	if c2sNode != nil {
		clientToServer, err := c2sNode.StartConnection("127.0.0.1", "user", "123456")
		if err != nil {
			log.Println("Unable to get TOTP from serverRouter", clientToServer)
			log.Fatal(err)
		}
		log.Println("Client -> Server TOTP exchange done:", clientToServer)
	}

	time.Sleep(300 * time.Millisecond)

	//Client to static
	if c2staticNode != nil {
		totp, err := c2staticNode.StartConnection("127.0.0.1", "user", "123456")
		if err != nil {
			log.Println("Unable to get TOTP from static server", totp)
			log.Fatal(err)
		}
		log.Println("Client -> Static TOTP exchange done:", totp)
	}
	time.Sleep(300 * time.Millisecond)

	//Server to Client
	if s2cNode != nil {
		serverToClient, err := s2cNode.StartConnection("127.0.0.1", "user", "123456")
		if err != nil {
			log.Println("Unable to get TOTP from clientRouter", serverToClient)
			log.Fatal(err)
		}
		log.Println("Server -> Client TOTP exchange done:", serverToClient)
	}
	time.Sleep(300 * time.Millisecond)

	//Server to Static Server
	if s2staticNode != nil {
		totp, err := s2staticNode.StartConnection("127.0.0.1", "user", "123456")
		if err != nil {
			log.Println("Unable to get TOTP from static", totp)
			log.Fatal(err)
		}
		log.Println("Server -> Static TOTP exchange done:", totp)
	}
	time.Sleep(300 * time.Millisecond)

	//Static to Client
	if static2cNode != nil {
		totp, err := static2cNode.StartConnection("127.0.0.1", "user", "123456")
		if err != nil {
			log.Println("Unable to get TOTP from client", totp)
			log.Fatal(err)
		}
		log.Println("Static -> Client TOTP exchange done:", totp)
	}
	time.Sleep(300 * time.Millisecond)

	//Static to Server
	if static2sNode != nil {
		totp, err := static2sNode.StartConnection("127.0.0.1", "user", "123456")
		if err != nil {
			log.Println("Unable to get TOTP from server", totp)
			log.Fatal(err)
		}
		log.Println("Static -> Server TOTP exchange done:", totp)
	}
	time.Sleep(300 * time.Millisecond)

	/*
		fmt.Println("Client TOTP Map: ")
		clientRouter.PrettyPrintTOTPMap()
		fmt.Println("Server TOTP Map: ")
		serverRouter.PrettyPrintTOTPMap()
		fmt.Println("Static TOTP Map: ")
		staticRouter.PrettyPrintTOTPMap()
	*/

	//Optional: Enable verbal output on both router
	clientRouter.Options.Verbal = true
	serverRouter.Options.Verbal = true
	staticRouter.Options.Verbal = true

	clientRouter.StartHeartBeat()
	serverRouter.StartHeartBeat()
	staticRouter.StartHeartBeat()

	//Show the client and server IP address after 2 heart beat cycles
	go func() {
		time.Sleep(11 * time.Second)
		fmt.Println("Client IP Address is: ", clientRouter.DeviceIpAddr.String())
		fmt.Println("Client's nearby nodes are: ", clientRouter.GetNeighbourNodes())
		fmt.Println("Server IP Address is: ", serverRouter.DeviceIpAddr.String())
		fmt.Println("Server's nearby nodes are: ", serverRouter.GetNeighbourNodes())
		fmt.Println("Static IP Address is: ", staticRouter.DeviceIpAddr.String())
		fmt.Println("Static's nearby nodes are: ", staticRouter.GetNeighbourNodes())
		//Disable the client http listener
		enableClientHTTP = false
		enableServerHTTP = false
	}()

	//Export the configuration to file

	go func() {
		time.Sleep(120 * time.Second)
		/*
			//Export client Router
			js, _ := clientRouter.ExportRouterToJSON()
			ioutil.WriteFile("clientRouter.json", []byte(js), 0777)

			//Export server router
			js, _ = serverRouter.ExportRouterToJSON()
			ioutil.WriteFile("serverRouter.json", []byte(js), 0777)
		*/
		//End all service router function
		clientRouter.Close()
		serverRouter.Close()

		time.Sleep(1 * time.Second)
		os.Exit(1)
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
