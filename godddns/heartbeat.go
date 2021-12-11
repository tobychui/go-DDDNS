package godddns

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xlzd/gotp"
)

/*
	HeartBeat.go

	This script handle the heartbeat and ip mapping update logic
	for the DDDNS process
*/

type HeartBeatPacket struct {
	NodeUUID string
	TOTP     string
	IPADDR   string
}

func (s *ServiceRouter) StartHeartBeat() {
	beatingInterval := s.Options.SyncInterval
	if beatingInterval <= 0 {
		//Use default value 10 seconds
		beatingInterval = 10
	}

	//Check if there is a previous heart beat routine running. Kill it if true
	if s.heartBeatTickerChannel != nil {
		s.heartBeatTickerChannel <- true
	}

	//Execute the initiation heart beat cycle
	s.ExecuteHeartBeatCycle()

	//Create a heart beat ticker of given interval
	ticker := time.NewTicker(time.Duration(beatingInterval) * time.Second)
	quit := make(chan bool)
	s.heartBeatTickerChannel = quit
	go func() {
		for {
			select {
			case <-ticker.C:
				s.ExecuteHeartBeatCycle()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *ServiceRouter) StopHeartBeat() {
	if s.heartBeatTickerChannel != nil {
		s.heartBeatTickerChannel <- true
	}
}

//HandleHeartBeatRequest handle the heartbeat request from other nodes
func (s *ServiceRouter) HandleHeartBeatRequest(w http.ResponseWriter, r *http.Request) {
	// Declare a new credential structure
	var payload HeartBeatPacket

	//Try to parse it into the required structure
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Validate the TOTP
	targetTotpSecret := ""
	for _, thisTOTPRecord := range s.TOTPMap {
		if thisTOTPRecord.RemoteUUID == payload.NodeUUID {
			targetTotpSecret = thisTOTPRecord.RecvTOTPSecret
		}
	}

	if targetTotpSecret == "" {
		//No record found, target UUID did not register on this node
		http.Error(w, "node UUID not registered", http.StatusUnauthorized)
		return
	}

	targetTotpResolver := gotp.NewDefaultTOTP(targetTotpSecret)
	isValidTotp := targetTotpResolver.Verify(payload.TOTP, int(time.Now().Unix()))

	if !isValidTotp {
		//Response to invalid TOTP
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("400 - Invalid TOTP"))
		return
	}

	//Get the node object from the NodeMap and updates its IP address
	for _, node := range s.NodeMap {
		if node.UUID == payload.NodeUUID {
			node.IpAddr = net.ParseIP(payload.IPADDR)
			break
		}
	}

	//Reply the IP address of the requesting node from this node's perspective
	w.Write([]byte(r.RemoteAddr))
}

/*
	ExecuteHeartBeatCycle will send a heartbeat signal to all registered nodes and update
	the current node's public / private IP address
*/
func (s *ServiceRouter) ExecuteHeartBeatCycle() {
	//Execute heartbeat on all connected nodes
	for _, node := range s.NodeMap {
		s.heartBeatToNode(node)
	}

	//Vote the correct ip address from what other nodes told us
	pubip, priip := s.VoteRouterIPAddr()

	//Use its public IP as this node IP, if public ip is not found (aka LAN cluster)
	//use private IP address instead
	var newIp net.IP
	if pubip.String() != "0.0.0.0" {
		newIp = pubip
	} else {
		newIp = priip
	}

	if newIp.String() != "" && newIp.String() != s.DeviceIpAddr.String() {
		//IP has changed.
		s.LastIpUpdateTime = time.Now().Unix()
	}
	s.DeviceIpAddr = newIp
	s.LastSyncTime = time.Now().Unix()
	log.Println(s)
}

//HeartBeatToNode execute a one-time heartbeat update to given node with matching UUID
func (s *ServiceRouter) HeartBeatToNode(nodeUUID string) error {
	targetNode := s.getNodeByUUID(nodeUUID)
	if targetNode == nil {
		return errors.New("node with given UUID not found")
	}

	return s.heartBeatToNode(targetNode)
}

//VoteRouterIPAddr will check all the IP addresses return from the network of nodes
//and decide what is the current router public and private IP address
func (s *ServiceRouter) VoteRouterIPAddr() (net.IP, net.IP) {
	privateIps := map[string]int{}
	publicIps := map[string]int{}
	//Create the key value pairs
	for _, node := range s.NodeMap {
		if node.ReflectedIP != "" {
			publicIps[node.ReflectedIP] = 0
		}

		if node.ReflectedPrivateIP != "" {
			privateIps[node.ReflectedPrivateIP] = 0
		}
	}

	//Count the number of pairs in the node map
	publicIpMaxCount := 0
	privateIpMaxCount := 0
	for _, node := range s.NodeMap {
		if node.ReflectedIP != "" {
			publicIps[node.ReflectedIP]++
			if publicIps[node.ReflectedIP] > publicIpMaxCount {
				publicIpMaxCount = publicIps[node.ReflectedIP]
			}
		}

		if node.ReflectedPrivateIP != "" {
			privateIps[node.ReflectedPrivateIP]++
			if privateIps[node.ReflectedPrivateIP] > privateIpMaxCount {
				privateIpMaxCount = privateIps[node.ReflectedPrivateIP]
			}
		}
	}

	//Extract the vote results for public and private ips
	votePublicIpResult := "0.0.0.0"
	votePrivateIpResult := "0.0.0.0"
	for ip, count := range publicIps {
		if count == publicIpMaxCount {
			votePublicIpResult = ip
			break
		}
	}

	for pip, count := range privateIps {
		if count == privateIpMaxCount {
			votePrivateIpResult = pip
			break
		}
	}

	//Prase the IP to return correct datatypes
	rpub := net.ParseIP(votePublicIpResult)
	if rpub == nil {
		rpub = net.ParseIP("0.0.0.0")
	}
	rpri := net.ParseIP(votePrivateIpResult)
	if rpri == nil {
		rpri = net.ParseIP("0.0.0.0")
	}
	return rpub, rpri
}

/*
	Internal Functions

*/

//getNodebyUUID return the node that with the given uuid, return nil if not found
func (s *ServiceRouter) getNodeByUUID(uuid string) *Node {
	for _, node := range s.NodeMap {
		if node.UUID == uuid {
			return node
		}
	}

	return nil
}

/*
	heartBeatToNode create an heartbeat signal to the target node and updates its address based on the
	DDDNS implementation. Updates will be written directly to the node object pointed by the poitner
*/
func (s *ServiceRouter) heartBeatToNode(node *Node) error {
	//Assemble the target node heartbeat endpoint
	reqEndpoint := node.ReflectedIP + ":" + strconv.Itoa(node.Port) + "/" + node.HeartbeatRelpath
	reqEndpoint = filepath.ToSlash(filepath.Clean(reqEndpoint))

	//Append protocol type
	if node.RequireHTTPS {
		reqEndpoint = "https://" + reqEndpoint
	} else {
		reqEndpoint = "http://" + reqEndpoint
	}

	log.Println(reqEndpoint)

	//Generate a TOTP for this node
	totp := gotp.NewDefaultTOTP(node.SendTotpSecret)
	token := totp.Now()

	//POST this node's IP address to the target node
	postBody, _ := json.Marshal(map[string]string{
		"NodeUUID": s.Options.DeviceUUID,
		"TOTP":     token,
		"IPADDR":   s.DeviceIpAddr.String(),
	})
	responseBody := bytes.NewBuffer(postBody)

	//Record last sync time
	node.lastSync = time.Now().Unix()

	//Create a POST request to the target node heartbeat endpoint
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Post(reqEndpoint, "application/json", responseBody)
	if err != nil {
		//Post failed, clear all the IP fields
		node.ReflectedIP = ""
		node.ReflectedPrivateIP = ""
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	//The returned body should contain this node's ip address as seen by the other node
	log.Println("Heartbeat reflected IP: ", string(body), err)
	reflectedIp := string(body) //This node IP as seens by the requested node
	reflectedIp = trimIpPort(reflectedIp)

	//Update node information
	node.lastOnline = node.lastSync

	if isPrivateIpString(reflectedIp) {
		node.ReflectedPrivateIP = reflectedIp
		node.ReflectedIP = ""
	} else {
		node.ReflectedPrivateIP = ""
		node.ReflectedIP = reflectedIp
	}

	return nil
}
