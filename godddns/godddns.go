package godddns

import (
	"net"
	"time"
)

type Node struct {
	UUID               string
	ConnectionEndpoint string
	HeartBeatEndpoint  string

	lastOnline int64
	lastSync   int64
	totpSecret string
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

func (s *ServiceRouter) AddNode(remoteUUID string, connectionEndpoint string, heartBeatEndpoint string) {
	s.NodeMap = append(s.NodeMap, &Node{
		UUID:               remoteUUID,
		ConnectionEndpoint: connectionEndpoint,
		HeartBeatEndpoint:  heartBeatEndpoint,
		lastOnline:         0,
		lastSync:           0,
		totpSecret:         "",
	})

}
