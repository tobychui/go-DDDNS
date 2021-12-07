package godddns

import (
	"log"
	"net/http"
	"time"
)

/*
	HeartBeat.go

	This script handle the heartbeat and ip mapping update logic
	for the DDDNS process
*/

func (s *ServiceRouter) StartHeartBeat(beatingInterval int) {
	if beatingInterval <= 0 {
		//Use default value 10 seconds
		beatingInterval = 10
	}

	//Check if there is a previous heart beat routing running. Kill it if true
	if s.heartBeatTickerChannel != nil {
		s.heartBeatTickerChannel <- true
	}

	//Execute the initiation heart beat cycle
	s.ExecuteHeartBeatCycle()

	//Create a heart beat ticker of given interval
	ticker := time.NewTicker(time.Duration(beatingInterval) * time.Second)
	quit := make(chan bool)
	s.heartBeatTickerChannel = quit
	go func() {
		for {
			select {
			case <-ticker.C:
				s.ExecuteHeartBeatCycle()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *ServiceRouter) StopHeartBeat() {
	if s.heartBeatTickerChannel != nil {
		s.heartBeatTickerChannel <- true
	}
}

func (s *ServiceRouter) HandleHeartBeatRequest(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (s *ServiceRouter) ExecuteHeartBeatCycle() {
	log.Println(s.Options.DeviceUUID, "Heartbeat executed")
}

func (s *ServiceRouter) HeartBeatToNode(nodeUUID string) {

}
