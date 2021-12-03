package godddns

import "net/http"

func (s *ServiceRouter) HandleHeartBeatRequest(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (s *ServiceRouter) ExecuteHeartBeatCycle() {

}

func (s *ServiceRouter) HeartBeatToNode(nodeUUID string) {

}

func (s *ServiceRouter) heartBeatToNode(nodeUUID string) {

}
