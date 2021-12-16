package godddns

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xlzd/gotp"
)

/*
	Sync.go

	This script handles the sync request from node
	that connection retries exceed max count
*/

type SyncRequestPackage struct {
	NodeUUID   string
	TOTP       string
	AskingUUID string
}

func (s *ServiceRouter) syncNodeAddress(node *Node) error {
	//Get the nodes that is recently updated
	latestUpdatedNodes := []*Node{}
	timeBaseline := time.Now().Unix() - 2*s.Options.SyncInterval
	for _, node := range s.NodeMap {
		if node.lastOnline > timeBaseline {
			//This node is newly updated
			latestUpdatedNodes = append(latestUpdatedNodes, node)
		}
	}

	if len(latestUpdatedNodes) == 0 {
		if s.Options.Verbal {
			log.Println("Unable to reach any nodes. Entering orphan mode.")
		}
		return errors.New("node in orphan mode")
	}

	//Randomly pick one from the node list
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	askingNode := latestUpdatedNodes[rand.Intn(len(latestUpdatedNodes))]

	//Ask the asking node for the target node's ip address
	newNodeIp, err := s.resolveNodeIpFromAskingNode(node, askingNode)
	if err != nil {
		return err
	}

	if newNodeIp.String() == node.IpAddr.String() {
		//IP didnt change as seen from the 3rd node. Ask another random node in next cycle
		if s.Options.Verbal {
			log.Println(node.UUID + " update not yet iterate to this node. Checking in next cycle.")
		}
	} else {
		//IP addr different. Update it and reset retry count
		node.IpAddr = newNodeIp
		node.retryCount = 0
	}
	return nil
}

func (s *ServiceRouter) handleSyncRequestByLostNode(w http.ResponseWriter, r *http.Request) {
	// Declare a new credential structure
	var payload SyncRequestPackage

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

	//Check if the asking node UUID exists in this node's registered node list

	//Reply the IP address of the requesting node from this node's perspective
	w.Write([]byte(r.RemoteAddr))
}

func (s *ServiceRouter) resolveNodeIpFromAskingNode(node *Node, askingNode *Node) (net.IP, error) {
	//Assemble the target node heartbeat endpoint
	reqEndpoint := node.IpAddr.String() + ":" + strconv.Itoa(askingNode.Port) + "/" + askingNode.RESTfulInterface + "?opr=s"
	reqEndpoint = filepath.ToSlash(filepath.Clean(reqEndpoint))

	//Append protocol type
	if node.RequireHTTPS {
		reqEndpoint = "https://" + reqEndpoint
	} else {
		reqEndpoint = "http://" + reqEndpoint
	}

	//Generate a TOTP for this node
	totp := gotp.NewDefaultTOTP(askingNode.SendTotpSecret)
	token := totp.Now()

	//POST the request asking for the target node
	postBody, _ := json.Marshal(map[string]string{
		"NodeUUID":   s.Options.DeviceUUID,
		"TOTP":       token,
		"AskingUUID": node.UUID,
	})
	responseBody := bytes.NewBuffer(postBody)

	//Create a POST request to the target node heartbeat endpoint
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Post(reqEndpoint, "application/json", responseBody)
	if err != nil {
		//Post failed, return the error
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.Println(string(body))
	return nil, nil
}
