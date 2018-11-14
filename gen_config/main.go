package main

import (
	"encoding/json"
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/SKF/go-enlight-sdk/grpc"
	"github.com/SKF/go-enlight-sdk/services/hierarchy"
	"github.com/SKF/go-utility/log"
)

// Config map[functional_location_name]map[asset_name]map[point_name]point_id
type Config map[string]map[string]map[string]string

var client hierarchy.HierarchyClient
var replacer = strings.NewReplacer(" ", "_")

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

// GenerateConfigFromParentID populate config recursivley
func GenerateConfigFromParentID(config Config, funcLoc string, asset string, parentID string) Config {
	childNodes, err := client.GetChildNodes(parentID)
	if err != nil {
		log.Error(err)
	}

	for _, childNode := range childNodes {
		log.WithField("node", childNode).Info("found childnode")
		if childNode.Type == "functional_location" {
			funcLoc = replacer.Replace(strings.ToLower(childNode.Label))
			config[funcLoc] = make(map[string]map[string]string)

		} else if childNode.Type == "asset" {
			asset = replacer.Replace(strings.ToLower(childNode.Label))
			config[funcLoc][asset] = make(map[string]string)
			config[funcLoc][asset]["__location_id__"] = parentID
			config[funcLoc][asset]["__asset_id__"] = childNode.Id

		} else {
			point := replacer.Replace(strings.ToLower(childNode.Label))
			config[funcLoc][asset][point] = childNode.Id

		}
		GenerateConfigFromParentID(config, funcLoc, asset, childNode.Id)
	}
	return config
}

func main() {
	// Default UUID is SKF DEV CENTER GOT
	var uuid string
	flag.StringVar(&uuid, "uuid", "fb98275e-330c-4b46-8dfa-785c3ddf2d8a", "Site UUID from Enlight")
	flag.Parse()

	client = DialHierarchy()
	defer client.Close()

	config := make(Config)
	config = GenerateConfigFromParentID(config, "", "", uuid)

	jsonData, err := json.Marshal(config)
	if err != nil {
		log.Error(err)
	}

	ex, err := os.Executable()
	if err != nil {
		log.Error(err)
	}
	exPath := filepath.Dir(ex)

	jsonFile, err := os.Create(path.Join(exPath, "./config.json"))
	if err != nil {
		log.Error(err)
	}
	defer jsonFile.Close()
	jsonFile.Write(jsonData)

	return
}
