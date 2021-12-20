# go-DDDNS
Go implementation of the Routing ID based Distributed Dynamic Domain Name Service

## Introduction

There is no such standard as DDDNS and this is not a DDNS protocol. I drafted this experiment myself to test if it is possible to create a clusters that "floats" on the floating IP assigned by the ISP or in a network environment that can only allow plug-and-play hosting.

The requirement of this project is as follows.

1. The software cannot get IP address from its NIC, or using any kind of commands that request OS to provide the IP address information (e.g. No ip a or ifconfig)
2. No external dependencies other than the connected nodes (e.g. no UPnP to ask router what is the current node IP, no online IP checking API)
3. No platform dependencies (aka it should not be only working on Linux)

All of the ip address information has to be come from its connected nodes and the packets they are sending each others, manual input is acceptable only during the initial setting up and the software itself should handle all the ip changes and map itself correctly in the cluster / node mesh.

If this project is proofed to be working and secure, this will be added to the ArozOS project as the fundamental section of its Clustering System.

## Usage

### Basics

```Go
import (
	godddns "github.com/tobychui/go-DDDNS/godddns"
)

func ValidateCred(username string, password string) bool{
    return (username == "user" && password == "password")
}

//Create a new node object
thisNode := godddns.NewServiceRouter(godddns.RouterOptions{
    DeviceUUID:   "thisNode",
    AuthFunction: ValidateCred,
    SyncInterval: 10,
})

//Start heartbeat to other nodes
thisNode.StartHeartBeat()

//Do a blocking loop
select {}
```

### Minimum Working Example 

This module require at least two nodes across network to work properly.  The following example assumed the following network conditions:

| Node ID    | IP Address    | Connection Endpoint |
| ---------- | ------------- | ------------------- |
| NAT Router | 192.168.0.1   | N/A                 |
| node1      | 192.168.0.100 | /godddns            |
| node2      | 192.168.0.101 | /godddns            |

The demo code showcase the node1's logic with go-DDDNS

```go
import (
	godddns "github.com/tobychui/go-DDDNS/godddns"
)

func ValidateCred(username string, password string) bool{
    //Implement your username and password check here
	return true
}

func main(){
    //Create new service router as node1 (this node)
		node1 = godddns.NewServiceRouter(godddns.RouterOptions{
            DeviceUUID:   "node1",
            AuthFunction: ValidateCred,
            SyncInterval: 10,
        })
        
        //Add node2 node into the client list
		node2 := clientRouter.NewNode(godddns.NodeOptions{
			NodeID:        "node2",
			Port:          8080,
			RESTInterface: "/godddns",
			RequireHTTPS:  false,
		})
		node1.AddNode(node2)
        
    	//Start connection listener at port 8080
        go func() {
            http.HandleFunc("/godddns", node1.HandleConnections)
            http.ListenAndServe(":8080", serverHandler)
        }()
        
    	//Start connection to node2, fill in the current node2 ip address and login credentials
    	totpSecret, err := node2.StartConnection("192.168.0.101", "username", "password")
    
    	//Start Heartbeat
        serviceRouter.StartHeartBeat()
    	
    	//Do a blocking loop
    	select {}
    
        //To end the node and unregister all nodes, call to
        //node1.Close()
}

```

Alternatively, you can perform testing to the module with three nodes spawned out from the same process but listen to different ports to test for implementation logic errors (but not runtime / network errors). See main.go in go-DDDNS repo for such testing demo. 

## Demo Videos

Two-nodes IP tracking demo

[![](https://img.youtube.com/vi/Qnpuubt70I4/0.jpg)](https://www.youtube.com/watch?v=Qnpuubt70I4)

Three-nodes IP change in one heartbeat cycle and synchronize from static node demo

[![](https://img.youtube.com/vi/UcgDCWygO2Q/0.jpg)](https://www.youtube.com/watch?v=UcgDCWygO2Q)



## License

MIT

(For now, please don't do anything with it, **especially don't use it in production**, however comments and ideas are welcomed via **issues**)

