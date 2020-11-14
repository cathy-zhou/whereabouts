package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"regexp"

	cnitypes "github.com/containernetworking/cni/pkg/types/current"
	"github.com/dougbtv/whereabouts/pkg/allocate"
	"github.com/dougbtv/whereabouts/pkg/logging"
	"github.com/dougbtv/whereabouts/pkg/storage"
	whereaboutstypes "github.com/dougbtv/whereabouts/pkg/types"
)

func main() {
	// walk through the list of files
	resultsDir := os.Getenv("WHEREABOUTS_RESULTSDIR")
	files, err := ioutil.ReadDir(resultsDir)
	if err != nil && err != os.ErrNotExist {
		logging.Errorf("Failed to list the files in %s, error: %v", resultsDir, err)
		os.Exit(1)
	}

	// The following regular expression isto match files with names
	// sriov-public-net-2941353d3bdc0284887261614d809a03685ca07dc753132ad28c118f3df37f77-net1
	// sriov-public-nd-sjc6o-02-net-b5679ae7dd8205464f2accb1a18c568d7b62d5a9d4b10801069d99efa0f00e90-net1
	// The huge alpha-numeric name is the pause container ID.
	r, _ := regexp.Compile(`sriov-public[a-z0-9-]*-net-([a-z0-9]+)-net[12]`)
	for _, file := range files {
		submatches := r.FindAllStringSubmatch(file.Name(), -1)
		if len(submatches) == 0 || len(submatches[0]) < 2 {
			logging.Debugf("Skipping the non sriov-public file - %s", file.Name())
			continue
		}
		containerID := submatches[0][1]
		contents, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", resultsDir, file.Name()))
		if err != nil {
			logging.Errorf("Failed to read the contents of this file - %s, error: %v",
				file.Name(), err)
			continue
		}
		logging.Debugf("Parsing the contents of file - %s to reserve the IP", file.Name())
		// The contents of the result structure looks somethign like this:
		// {
		//  "cniVersion": "0.4.0",
		//  "interfaces": [
		//    {
		//      "name": "net1",
		//      "mac": "02:00:00:2e:3f:75",
		//      "sandbox": "/proc/21545/ns/net"
		//    }
		//  ],
		//  "ips": [
		//    {
		//      "version": "4",
		//      "interface": 0,
		//      "address": "24.51.17.125/29",
		//      "gateway": "24.51.17.121"
		//    }
		//  ],
		//  "routes": [
		//    {
		//      "dst": "0.0.0.0/0"
		//    }
		//  ],
		//  "dns": {}
		// }
		ipamResults := &cnitypes.Result{}
		if err := json.Unmarshal(contents, ipamResults); err != nil {
			logging.Errorf("Failed while unmarshalling the file contents %s into cnitypes.Result structure, error: %v",
				contents, err)
			continue
		}
		addressToReserve := ipamResults.IPs[0].Address.String()
		reserveIP, reserveCIDR, err := net.ParseCIDR(addressToReserve)
		if err != nil {
			logging.Errorf("Failed while parsing the address %s to be reserved, error: %v", addressToReserve, err)
			continue
		}
		networkAddress := net.ParseIP(reserveCIDR.IP.String())
		firstIP, lastIP, err := allocate.GetIPRange(networkAddress, *reserveCIDR)
		if err != nil {
			logging.Errorf("Failed while determining the first and last ip from the range %s, error: %v",
				reserveCIDR.String(), err)
			continue
		}

		ipamConfig := whereaboutstypes.IPAMConfig{
			Range:      reserveCIDR.String(),
			RangeStart: firstIP,
			RangeEnd:   lastIP,
			Addresses: []whereaboutstypes.Address{
				{
					AddressStr: reserveIP.String(),
					Address:    net.IPNet{IP: reserveIP, Mask: reserveCIDR.Mask},
				},
			},
			Datastore: whereaboutstypes.DatastoreKubernetes,
			Kubernetes: whereaboutstypes.KubernetesConfig{
				KubeConfigPath: os.Getenv("WHEREABOUTS_KUBECONFIG"),
			},
		}
		logging.Debugf("Reserving IP - %s in Address Range - %s with firstIP - %s and lastIP - %s",
			reserveIP, reserveCIDR, firstIP, lastIP)
		_, err = storage.IPManagement(whereaboutstypes.Reserve, ipamConfig, containerID)
		if err != nil {
			logging.Errorf("Failed to reserve the IP %s from the range %s, error: %v",
				reserveIP, reserveCIDR, err)
			continue
		}
		logging.Debugf("Successfully reserved IP - %s", reserveIP)
	}
}
