package godddns

import (
	"net"
	"path/filepath"
	"time"
)

type Node struct {
	UUID              string //The UUID of the target Node
	IpAddr            string //The IP address of the Node
	Port              int    //The port for connection
	ConnectionRelpath string //The relative path for Establish connection
	HeartbeatRelpath  string //The relative path for Heartbeat connection
	ReflectedIP       string //The IP address reflected by the other node
	lastOnline        int64  //Last time this node is connectable
	lastSync          int64  //Last time this device tries to conenct this node
	totpSecret        string //The TOTPSecret for sending message
}

type TOTPRecord struct {
	RemoteUUID string //The remote node ID where this TOTP was sent to
	TOTPSecret string //The TOTP secret assigned to this node
}

type RouterOptions struct {
	DeviceUUID   string                    //The UUID of this device
	AuthFunction func(string, string) bool //Check if the authentication is correct based on username and password
	SyncInterval int64                     //Sync interval in seconds
}

type ServiceRouter struct {
	NodeMap          []*Node
	TOTPMap          []*TOTPRecord
	Options          *RouterOptions
	DeviceIpAddr     net.IP
	LastIpUpdateTime int64
	LastSyncTime     int64
}

func NewServiceRouter(options RouterOptions) *ServiceRouter {
	return &ServiceRouter{
		NodeMap:          []*Node{},
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
		IpAddr:            "",
		Port:              port,
		ConnectionRelpath: filepath.ToSlash(filepath.Clean(connectionRelativePath)),
		HeartbeatRelpath:  filepath.ToSlash(filepath.Clean(connectionRelativePath)),
		ReflectedIP:       "",
		lastOnline:        0,
		lastSync:          0,
		totpSecret:        "",
	}
}

//Add the node to this router
func (s *ServiceRouter) AddNode(node *Node) {
	s.NodeMap = append(s.NodeMap, node)
}

//Set the node's TOTPSecret by external database
func (n *Node) SetNodeTOTPSecret(totpSecret string) {
	n.totpSecret = totpSecret
}

//Extract the node's TOTPSecret
func (n *Node) ExtractTOTPSecret() string {
	return n.totpSecret
}
