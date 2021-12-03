package godddns

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/xlzd/gotp"
)

/*
	GoDDDNS Connection Setup Script

	This script setup and connect two nodes with given information
*/

type Credential struct {
	NodeUUID string //The remote node UUID
	Username string //The username that the account is using
	Password string //The password that the account is using
}

/*
	StartConnection
	Establish connection to a new node using a given UUID
	A node must be registered with AddNode first before StartConenction can be called
*/
func (s *ServiceRouter) StartConnection(targetNodeUUID string, username string, password string) (string, error) {
	//Look for the target node
	var targetNode *Node = nil
	for _, node := range s.NodeMap {
		if node.UUID == targetNodeUUID {
			//This is the target node
			thisNode := node
			targetNode = thisNode
		}
	}

	postBody, _ := json.Marshal(map[string]string{
		"NodeUUID": s.Options.DeviceUUID,
		"Username": username,
		"Password": password,
	})
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(targetNode.ConnectionEndpoint, "application/json", responseBody)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	sb := string(body)
	resp.Body.Close()
	return sb, nil
}

/*
	HandleConnectionEstablishResponse
	Handle incoming connection and generate & return TOTP for the connecting node
*/
func (s *ServiceRouter) HandleConnectionEstablishResponse(w http.ResponseWriter, r *http.Request) {
	// Declare a new credential structure
	var cred Credential

	//Try to parse it into the required structure
	err := json.NewDecoder(r.Body).Decode(&cred)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	//Validate the credential
	fmt.Println(cred)

	//Generate TOTP
	totpSecret := gotp.RandomSecret(32)

	//Write TOTP Secret to map
	s.TOTPMap = append(s.TOTPMap, &TOTPRecord{
		RemoteUUID: cred.NodeUUID,
		TOTPSecret: totpSecret,
	})

	//Return TOTP to request client
	w.Write([]byte(totpSecret))
}
