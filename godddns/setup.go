package godddns

import (
	"encoding/json"
	"log"
	"net/http"

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
	HandleConnectionEstablishResponse
	Handle incoming connection and generate & return TOTP for the connecting node
*/
func (s *ServiceRouter) handleConnectionEstablishResponse(w http.ResponseWriter, r *http.Request) {
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

	if s.Options.Verbal {
		log.Println(cred.NodeUUID + " has established connection with this node(" + s.Options.DeviceUUID + ")")
	}

	//Return TOTP to request client
	w.Write(result)
}
