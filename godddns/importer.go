package godddns

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

/*
	Importer

	This script handle the import and export of the router object

*/

//NewRouterFromJSON create a new router object from JSON string
//Notes that the newly created service router is still has its auth function missing
//the authentication function has to be injected after the router is returned
func NewRouterFromJSON(jsonConfig string) (*ServiceRouter, error) {
	newRouter := ServiceRouter{}
	err := json.Unmarshal([]byte(jsonConfig), &newRouter)
	if err != nil {
		return nil, err
	}

	//Fill the parent object for all nodes
	for _, registerNodes := range newRouter.NodeMap {
		registerNodes.parent = &newRouter
	}

	if len(newRouter.NodeMap) == 0 && newRouter.Options.Verbal {
		log.Println(newRouter.Options.DeviceUUID + " config has no 0 registered node!!!")
	}

	newRouter.IpChangeEventListener = nil

	return &newRouter, nil
}

//NewRouterFromJSONFile create a new router from file contianing json string
func NewRouterFromJSONFile(filename string) (*ServiceRouter, error) {
	fileContent, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return NewRouterFromJSON(string(fileContent))
}

//Inject an auth function into an imported service router
func (s *ServiceRouter) InjectAuthFunction(authFunction func(string, string) bool) {
	s.Options.AuthFunction = authFunction
}

//Export a service router to JSON string
func (s *ServiceRouter) ExportRouterToJSON() (string, error) {
	js, err := json.MarshalIndent(s, "", " ")
	return string(js), err
}
