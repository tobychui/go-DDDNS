package godddns

import (
	"errors"
	"net"
	"path/filepath"
	"time"
)

type Node struct {
	UUID               string //The UUID of the target Node
	IpAddr             net.IP //The IP address of the Node
	Port               int    //The port for connection
	ConnectionRelpath  string //The relative path for Establish connection
	HeartbeatRelpath   string //The relative path for Heartbeat connection
	ReflectedIP        string //The IP address reflected by the other node
	ReflectedPrivateIP string //The IP address reflected by local nodes, should be LAN address

	lastOnline     int64  //Last time this node is connectable
	lastSync       int64  //Last time this device tries to conenct this node
	RequireHTTPS   bool   //The connection to the node must pass through HTTPS
	SendTotpSecret string //The TOTPSecret for sending message
}

type TOTPRecord struct {
	RemoteUUID     string //The remote node ID where this TOTP was sent to
	RecvTOTPSecret string //The TOTP secret assigned to this node
}

type RouterOptions struct {
	DeviceUUID   string                    //The UUID of this device
	AuthFunction func(string, string) bool `json:"-"` //Check if the authentication is correct based on username and password
	SyncInterval int64                     //Sync interval in seconds
}

type ServiceRouter struct {
	NodeMap          []*Node
	TOTPMap          []*TOTPRecord
	Options          *RouterOptions
	DeviceIpAddr     net.IP
	LastIpUpdateTime int64
	LastSyncTime     int64

	heartBeatTickerChannel chan bool
}

func NewServiceRouter(options RouterOptions) *ServiceRouter {
	return &ServiceRouter{
		NodeMap:          []*Node{},
		TOTPMap:          []*TOTPRecord{},
		Options:          &options,
		DeviceIpAddr:     nil,
		LastIpUpdateTime: time.Now().Unix(),
		LastSyncTime:     0,
	}
}

//Create a New Node based on remoteUUID, conencting endpoint and heart beat endpoint
func (s *ServiceRouter) NewNode(remoteUUID string, port int, connectionRelativePath string, heartBeatRelativePath string) *Node {
	return &Node{
		UUID:              remoteUUID,
		ReflectedIP:       "",
		Port:              port,
		ConnectionRelpath: filepath.ToSlash(filepath.Clean(connectionRelativePath)),
		HeartbeatRelpath:  filepath.ToSlash(filepath.Clean(heartBeatRelativePath)),

		lastOnline:     0,
		lastSync:       0,
		SendTotpSecret: "",
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

//Add the node to this router
func (s *ServiceRouter) NodeRegistered(nodeUUID string) bool {
	for _, node := range s.NodeMap {
		if node.UUID == nodeUUID {
			return true
		}
	}
	return false
}
