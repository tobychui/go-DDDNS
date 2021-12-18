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
)

/*
	StartConnection
	Establish connection to a new node using a given UUID
	A node must be registered with AddNode first before StartConenction can be called
*/
func (n *Node) StartConnection(initIPAddr string, username string, password string) (string, error) {
	//Check if the service router was correctly set-up
	if n.parent.Options.AuthFunction == nil {
		return "", errors.New("this service router does not contain a valid auth function")
	}

	//Use this ip address as its initial IP address
	n.IpAddr = net.ParseIP(initIPAddr)

	postBody, _ := json.Marshal(map[string]string{
		"NodeUUID": n.parent.Options.DeviceUUID,
		"Username": username,
		"Password": password,
	})
	responseBody := bytes.NewBuffer(postBody)
	protocol := "http://"
	if n.RequireHTTPS {
		protocol = "https://"
	}

	reqEndpoint := initIPAddr + ":" + strconv.Itoa(n.Port) + "/" + n.RESTfulInterface + "?opr=c"
	reqEndpoint = protocol + filepath.ToSlash(filepath.Clean(reqEndpoint))

	resp, err := http.Post(reqEndpoint, "application/json", responseBody)
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

	if n.ReflectedIP == "" {
		//Initialization
		n.ReflectedIP = reflectedIP
	}

	if isPrivateIpString(n.ReflectedIP) {
		n.ReflectedPrivateIP = reflectedIP
	} else {
		n.ReflectedIP = reflectedIP
	}

	n.SendTotpSecret = payload.TOTPSecret
	if n.parent.Options.Verbal {
		log.Println(n.parent.Options.DeviceUUID, " received payload for handshake: ", payload)
	}

	n.retryUsername = username
	n.retryPassword = password

	return payload.TOTPSecret, nil
}

/*
	EndConnection

	This function stop and remove the TOTP Map from the list
*/

func (n *Node) EndConnection() error {
	//Check if node is connected
	if n.parent.totpMapExists(n.UUID) < 0 {
		return errors.New("node is not conennected")
	}

	newTotpMap := []*TOTPRecord{}
	for _, record := range n.parent.TOTPMap {
		if record.RemoteUUID != n.UUID {
			newTotpMap = append(newTotpMap, record)
		}
	}
	n.parent.TOTPMap = newTotpMap
	if n.parent.Options.Verbal {
		log.Println("Node " + n.UUID + " disconnected")
	}
	return nil
}
