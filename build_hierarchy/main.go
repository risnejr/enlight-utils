package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/SKF/go-enlight-sdk/grpc"
	"github.com/SKF/go-enlight-sdk/services/hierarchy"
	"github.com/SKF/go-utility/log"
	grpcapi "github.com/SKF/proto/hierarchy"
)

// Config map[functional_location_name]map[asset_name]map[point_name]point_id
type Config map[string]map[string]map[string]string

// DialHierarchy dials enlights grpc server and returns a client
func DialHierarchy() hierarchy.HierarchyClient {
	HOST := "grpc.hierarchy.enlight.skf.com"
	PORT := "50051"

	ex, err := os.Executable()
	if err != nil {
		log.Error(err)
	}
	exPath := filepath.Dir(ex)

	CLIENTCRT := path.Join(exPath, "./certs/hierarchy/client.crt")
	CLIENTKEY := path.Join(exPath, "./certs/hierarchy/client.key")
	CACRT := path.Join(exPath, "./certs/hierarchy/ca.crt")

	client := hierarchy.CreateClient()
	transportOption, err := grpc.WithTransportCredentials(
		HOST, CLIENTCRT, CLIENTKEY, CACRT,
	)
	if err != nil {
		log.
			WithError(err).
			WithField("serverName", HOST).
			WithField("clientCrt", CLIENTCRT).
			WithField("clientKey", CLIENTKEY).
			WithField("caCert", CACRT).
			Error("grpc.WithTransportCredentials")
		return nil
	}

	err = client.Dial(
		HOST, PORT,
		transportOption,
		grpc.WithBlock(),
		grpc.FailOnNonTempDialError(true),
	)
	if err != nil {
		log.
			WithError(err).
			WithField("host", HOST).
			WithField("port", PORT).
			Error("client.Dial")
		return nil
	}

	if err = client.DeepPing(); err != nil {
		log.WithError(err).Error("client.DeepPing")
		return nil
	}

	return client
}

// GenerateCustomAsset requires funcLoc and asset of your choice and siteID which is the uuid from an already existing root node
func GenerateCustomAsset(funcLocName string, assetName string, siteID string) {
	client := DialHierarchy()

	ex, err := os.Executable()
	if err != nil {
		log.Error(err)
	}
	exPath := filepath.Dir(ex)

	config := make(Config)
	jsonFile, err := os.Open(path.Join(exPath, "./config.json"))
	if err != nil {
		log.Error(err)
	}
	defer jsonFile.Close()
	data, _ := ioutil.ReadAll(jsonFile)
	err = json.Unmarshal(data, &config)

	// Hard coded ID from system user
	userID := "7f8ba22a-7b4a-470a-a6e9-61906b2dddc5"

	// Only create the functional location if it doesn't exist in config
	var funcLocID string
	if len(config[ToConfig(funcLocName)]) == 0 {
		funcLocNode := grpcapi.Node{
			Label:   funcLocName,
			Type:    "functional_location",
			SubType: "functional_location"}
		funcLoc := grpcapi.SaveNodeInput{Node: &funcLocNode, ParentId: siteID, UserId: userID}
		funcLocID, err = client.SaveNode(funcLoc)
		if err != nil {
			log.Error(err)
		}
	} else {
		for _, assetInfo := range config[ToConfig(funcLocName)] {
			funcLocID = assetInfo["__location_id__"]
			break
		}
	}

	// Only generate hierarchy if asset doesn't exist
	if len(config[ToConfig(funcLocName)][ToConfig(assetName)]) == 0 {
		assetNode := grpcapi.Node{
			Label:     assetName,
			Type:      "asset",
			SubType:   "asset",
			AssetNode: &grpcapi.AssetNode{Criticality: "criticality_c"}}
		asset := grpcapi.SaveNodeInput{Node: &assetNode, ParentId: funcLocID, UserId: userID}
		assetID, err := client.SaveNode(asset)
		if err != nil {
			log.Error(err)
		}

		temperature := grpcapi.InspectionPoint{ValueType: 0, NumericUnit: "C"}
		temperatureNode := grpcapi.Node{
			Label:           "Temperature",
			Type:            "inspection_point",
			SubType:         "inspection_point",
			InspectionPoint: &temperature}
		humidity := grpcapi.InspectionPoint{ValueType: 0, NumericUnit: "%"}
		humidityNode := grpcapi.Node{
			Label:           "Humidity",
			Type:            "inspection_point",
			SubType:         "inspection_point",
			InspectionPoint: &humidity}
		pressure := grpcapi.InspectionPoint{ValueType: 0, NumericUnit: "hPa"}
		pressureNode := grpcapi.Node{
			Label:           "Pressure",
			Type:            "inspection_point",
			SubType:         "inspection_point",
			InspectionPoint: &pressure}
		gas := grpcapi.InspectionPoint{ValueType: 0, NumericUnit: "Ohm"}
		gasNode := grpcapi.Node{
			Label:           "Gas",
			Type:            "inspection_point",
			SubType:         "inspection_point",
			InspectionPoint: &gas}
		inspectionPoints := grpcapi.Nodes{Nodes: []*grpcapi.Node{&gasNode, &pressureNode, &humidityNode, &temperatureNode}}
		for _, ipNode := range inspectionPoints.Nodes {
			inspectionPoint := grpcapi.SaveNodeInput{Node: ipNode, ParentId: assetID, UserId: userID}
			_, err := client.SaveNode(inspectionPoint)
			if err != nil {
				log.Error(err)
			}
		}
	} else {
		fmt.Printf("Asset \"%s\" does already exist under \"%s\"", assetName, funcLocName)
	}

	return
}

// ToTitle ...
func ToTitle(name string) string {
	insertSpace := strings.Replace(name, "_", " ", -1)
	newName := strings.Title(insertSpace)
	return newName
}

// ToConfig ...
func ToConfig(name string) string {
	insertUnderscore := strings.Replace(name, " ", "_", -1)
	newName := strings.ToLower(insertUnderscore)
	return newName
}

func main() {
	// Default UUID is SKF DEV CENTER GOT
	var uuid string
	flag.StringVar(&uuid, "uuid", "fb98275e-330c-4b46-8dfa-785c3ddf2d8a", "Site UUID from Enlight")
	flag.Parse()

	hostname, err := os.Hostname()
	if err != nil {
		log.Error(err)
	}
	names := strings.Split(hostname, "-")
	funcLocName := ToTitle(names[0])
	assetName := ToTitle(names[1])

	// SKF DEV CENTER GOT hardcoded uuid
	GenerateCustomAsset(funcLocName, assetName, uuid)

	return
}
