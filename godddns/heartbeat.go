package godddns

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
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

	//Check if there is a previous heart beat routing running. Kill it if true
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
			targetTotpSecret = thisTOTPRecord.TOTPSecret
		}
	}

	if targetTotpSecret == "" {
		//No record found, target UUID did not register on this node
		http.Error(w, "node UUID not registered", http.StatusUnauthorized)
		return
	}

	targetTotpResolver := gotp.NewDefaultTOTP(targetTotpSecret)
	isValidTotp := targetTotpResolver.Verify(payload.TOTP, int(time.Now().Unix()))

	//Reply the IP address of the requesting node from this node's perspective
	log.Println(s.Options.DeviceUUID+" Heart Beat Received from "+r.RemoteAddr, payload, isValidTotp)
	w.Write([]byte(r.RemoteAddr))
}

/*
	ExecuteHeartBeatCycle will send a heartbeat signal to all registered nodes and update
	the current node's public / private IP address
*/
func (s *ServiceRouter) ExecuteHeartBeatCycle() {
	log.Println(s.Options.DeviceUUID, "Heartbeat executed")
	for _, node := range s.NodeMap {
		s.heartBeatToNode(node)
	}

	s.VoteRouterIPAddr()
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
//and decide what is the current router (public) ip address
func (s *ServiceRouter) VoteRouterIPAddr() {

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
	if node.requireHTTPS {
		reqEndpoint = "https://" + reqEndpoint
	} else {
		reqEndpoint = "http://" + reqEndpoint
	}

	log.Println(reqEndpoint)

	//Generate a TOTP for this node
	totp := gotp.NewDefaultTOTP(node.totpSecret)
	token := totp.Now()

	//POST this node's IP address to the target node
	postBody, _ := json.Marshal(map[string]string{
		"NodeUUID": s.Options.DeviceUUID,
		"TOTP":     token,
		"IPADDR":   string(s.DeviceIpAddr),
	})
	responseBody := bytes.NewBuffer(postBody)

	//Create a POST request to the target node heartbeat endpoint
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Post(reqEndpoint, "application/json", responseBody)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	//The returned body should contain this node's ip address as seen by the other node
	log.Println(resp, string(body), err)
	//Update node information

	return nil
}
