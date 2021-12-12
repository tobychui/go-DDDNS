package godddns

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/xlzd/gotp"
)

/*
	GoDDDNS Connection Setup Script

	This script setup and connect two nodes with given information
*/

//Send by active registration node
type Credential struct {
	NodeUUID string //The remote node UUID
	Username string //The username that the account is using
	Password string //The password that the account is using
}

//Return from registrated node
type TOTPPayload struct {
	TOTPSecret   string
	ReflectionIP string
}

/*
	StartConnection
	Establish connection to a new node using a given UUID
	A node must be registered with AddNode first before StartConenction can be called
*/
func (s *ServiceRouter) StartConnection(targetNodeUUID string, initIPAddr string, useHTTPS bool, username string, password string) (string, error) {
	//Look for the target node
	targetNode := s.getNodeByUUID(targetNodeUUID)
	if targetNode == nil {
		return "", errors.New("node not registered")
	}

	//Check if the service router was correctly set-up
	if s.Options.AuthFunction == nil {
		return "", errors.New("this service router does not contain a valid auth function")
	}

	//Use this ip address as its initial IP address
	targetNode.IpAddr = net.ParseIP(initIPAddr)

	postBody, _ := json.Marshal(map[string]string{
		"NodeUUID": s.Options.DeviceUUID,
		"Username": username,
		"Password": password,
	})
	responseBody := bytes.NewBuffer(postBody)
	protocol := "http://"
	if useHTTPS {
		protocol = "https://"
	}
	resp, err := http.Post(protocol+initIPAddr+":"+strconv.Itoa(targetNode.Port)+targetNode.ConnectionRelpath, "application/json", responseBody)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	payload := TOTPPayload{}
	err = json.Unmarshal(body, &payload)
	resp.Body.Close()
	if err != nil {
		return string(body), err
	}

	reflectedIP := trimIpPort(payload.ReflectionIP)

	if targetNode.ReflectedIP == "" {
		//Initialization
		targetNode.ReflectedIP = reflectedIP
	}

	if isPrivateIpString(targetNode.ReflectedIP) {
		targetNode.ReflectedPrivateIP = reflectedIP
	} else {
		targetNode.ReflectedIP = reflectedIP
	}

	targetNode.SendTotpSecret = payload.TOTPSecret
	targetNode.RequireHTTPS = useHTTPS
	if s.Options.Verbal {
		log.Println(s.Options.DeviceUUID, " received payload for handshake: ", payload)
	}

	return payload.TOTPSecret, nil
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
	if !s.Options.AuthFunction(cred.Username, cred.Password) {
		//Unauthorized
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Incorrect username or password"))
		return
	}

	//Check if the node TOTP already exists
	nodeTotpRecordPosition := s.totpMapExists(cred.NodeUUID)
	if nodeTotpRecordPosition >= 0 {
		//Node already registered. Remove the previous TOTP record
		s.TOTPMap[nodeTotpRecordPosition] = s.TOTPMap[len(s.TOTPMap)-1]
		s.TOTPMap = s.TOTPMap[:len(s.TOTPMap)-1]
	}

	//Generate TOTP
	totpSecret := gotp.RandomSecret(8)

	//Write TOTP Secret to map
	s.TOTPMap = append(s.TOTPMap, &TOTPRecord{
		RemoteUUID:     cred.NodeUUID,
		RecvTOTPSecret: totpSecret,
	})

	//Construct response
	payload := TOTPPayload{
		TOTPSecret:   totpSecret,
		ReflectionIP: r.RemoteAddr,
	}

	result, _ := json.Marshal(payload)

	//Return TOTP to request client
	w.Write(result)
}
