package godddns

import (
	"encoding/json"
	"log"
)

/*
	Importer

	This script handle the import and export of the router object

*/

/*
func NewRouterFromJSON(jsonConfig string) (*ServiceRouter, error) {

	return nil
}

func NewRouterFromJSONFile(filename string) (*ServiceRouter, error) {

	return nil
}
*/

func (s *ServiceRouter) ExportRouterToJSON() (string, error) {
	js, err := json.MarshalIndent(s, "", " ")
	log.Println(string(js), err)
	return string(js), err
}
