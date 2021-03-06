package godddns

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type Node struct {
	UUID               string //The UUID of the target Node
	IpAddr             net.IP //The IP address of the Node
	Port               int    //The port for connection
	RESTfulInterface   string //The RESTFUL request interface
	ReflectedIP        string //The IP address reflected by the other node
	ReflectedPrivateIP string //The IP address reflected by local nodes, should be LAN address
	RequireHTTPS       bool   //The connection to the node must pass through HTTPS
	SendTotpSecret     string //The TOTPSecret for sending message

	lastOnline    int64          //Last time this node is connectable
	lastSync      int64          //Last time this device tries to conenct this node
	retryCount    int64          //The number of retries done on this node
	retryUsername string         //The username for retry
	retryPassword string         //The password for retry
	publicKey     []byte         //The public key of this node
	parent        *ServiceRouter `json:"-"` //The service router that this node belongs to
}

//New Node Options
type NodeOptions struct {
	NodeID        string //The UUID of this node
	Port          int    //The connection port for this node
	RESTInterface string //The RESTFUL request interface
	RequireHTTPS  bool   //Use HTTPS for this node
}

type TOTPRecord struct {
	RemoteUUID     string //The remote node ID where this TOTP was sent to
	RecvTOTPSecret string //The TOTP secret assigned to this node
}

type RouterOptions struct {
	DeviceUUID   string                    //The UUID of this device
	AuthFunction func(string, string) bool `json:"-"` //Check if the authentication is correct based on username and password
	SyncInterval int64                     //Sync interval in seconds
	Verbal       bool                      //Enable verbal output
}

type ServiceRouter struct {
	NodeMap                    []*Node
	TOTPMap                    []*TOTPRecord
	Options                    *RouterOptions
	DeviceIpAddr               net.IP
	LastIpUpdateTime           int64
	LastSyncTime               int64
	ConnectionRetryWaitTimeMin int
	ConnectionRetryWaitTimeMax int
	IpChangeEventListener      func(net.IP) `json:"-"`

	heartBeatTickerChannel chan bool
}

var (
	heartBeatRetryCount int64 = 3 //Heartbeat will change to sync mode after this retry conunt is reached
)

func NewServiceRouter(options RouterOptions) *ServiceRouter {
	return &ServiceRouter{
		NodeMap:                    []*Node{},
		TOTPMap:                    []*TOTPRecord{},
		Options:                    &options,
		DeviceIpAddr:               nil,
		LastIpUpdateTime:           time.Now().Unix(),
		LastSyncTime:               0,
		ConnectionRetryWaitTimeMin: 10,
		ConnectionRetryWaitTimeMax: 120,
		IpChangeEventListener:      nil,
	}
}

//Create a basic request router
func (s *ServiceRouter) HandleConnections(w http.ResponseWriter, r *http.Request) {
	oprType := r.URL.Query().Get("opr")
	if oprType == "c" {
		//Connection Request
		s.handleConnectionEstablishResponse(w, r)
	} else if oprType == "h" {
		//Heartbeat Request
		s.handleHeartBeatRequest(w, r)
	} else if oprType == "s" {
		//Sync Request
		s.handleSyncRequestByLostNode(w, r)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("400 - Bad Request"))
	}
}

//Create a New Node based on remoteUUID, conencting endpoint and heart beat endpoint
func (s *ServiceRouter) NewNode(options NodeOptions) *Node {
	return &Node{
		UUID:             options.NodeID,
		ReflectedIP:      "",
		Port:             options.Port,
		RESTfulInterface: filepath.ToSlash(filepath.Clean(options.RESTInterface)),
		RequireHTTPS:     options.RequireHTTPS,
		SendTotpSecret:   "",

		lastOnline: 0,
		lastSync:   0,
		retryCount: 0,
		parent:     s,
	}
}

//Add the node to this router
func (s *ServiceRouter) AddNode(node *Node) error {
	if s.NodeRegistered(node.UUID) {
		return errors.New("node already registered")
	}
	s.NodeMap = append(s.NodeMap, node)
	return nil
}

//Remove the node with given UUID
func (s *ServiceRouter) RemoveNode(nodeUUID string) error {
	if !s.NodeRegistered(nodeUUID) {
		return errors.New("node with given UUID not exists")
	}
	newNodeMap := []*Node{}
	newTotpMap := []*TOTPRecord{}

	for _, thisNode := range s.NodeMap {
		if thisNode.UUID != nodeUUID {
			newNodeMap = append(newNodeMap, thisNode)
		}
	}

	for _, thisNode := range s.TOTPMap {
		if thisNode.RemoteUUID != nodeUUID {
			newTotpMap = append(newTotpMap, thisNode)
		}
	}

	s.NodeMap = newNodeMap
	s.TOTPMap = newTotpMap
	return nil
}

//Add the node to this router
func (s *ServiceRouter) NodeRegistered(nodeUUID string) bool {
	nodeUUID = strings.TrimSpace(nodeUUID)
	for _, node := range s.NodeMap {
		if strings.TrimSpace(node.UUID) == nodeUUID {
			return true
		}
	}
	return false
}

func (s *ServiceRouter) NodeConnected(nodeUUID string) bool {
	targetNode := s.getNodeByUUID(nodeUUID)
	if targetNode == nil {
		return false
	}

	return s.totpMapExists(targetNode.UUID) >= 0
}

func (s *ServiceRouter) GetNodeByUUID(nodeUUID string) (*Node, error) {
	targetNode := s.getNodeByUUID(nodeUUID)
	if targetNode == nil {
		return nil, errors.New("node not found")
	} else {
		return targetNode, nil
	}
}

func (s *ServiceRouter) GetNodeIP(nodeUUID string) net.IP {
	targetNode := s.getNodeByUUID(nodeUUID)
	return targetNode.IpAddr
}

func (s *ServiceRouter) GetNeighbourNodes() []string {
	nodeUUIDs := []string{}
	for _, node := range s.NodeMap {
		nodeUUIDs = append(nodeUUIDs, node.UUID)
	}

	return nodeUUIDs
}

func (s *ServiceRouter) Close() {
	//Stop Heartbeat
	s.StopHeartBeat()

	//Disconnect all nodes
	for _, node := range s.NodeMap {
		node.EndConnection()
	}
}

func (s *ServiceRouter) PrettyPrintTOTPMap() {
	for _, totpR := range s.TOTPMap {
		fmt.Println(totpR.RemoteUUID + ": " + totpR.RecvTOTPSecret)
	}
}
