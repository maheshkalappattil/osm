package catalog

import(
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net"
	"time"

	"github.com/openservicemesh/osm/pkg/witesand"
)

func (mc *MeshCatalog) witesandHttpServerAndClient() {
	go mc.witesandHttpServer()
	go mc.witesandHttpClient()
}

func (mc *MeshCatalog) witesandHttpServer() {
	// GET local gatewaypods, also learn remote OSM clusterID and IP
	http.HandleFunc("/localgatewaypods", mc.GetLocalGatewayPods) // inter OSM

	// GET handlers
	http.HandleFunc("/allgatewaypods", mc.GetAllGatewayPods) // from waves
	http.HandleFunc("/endpoints", mc.LocalEndpoints) // inter OSM

	// POST handler
	http.HandleFunc("/apigroupMap", mc.ApigroupMapping)

	http.ListenAndServe(":" + witesand.HttpServerPort , nil)
}

func (mc *MeshCatalog) witesandHttpClient() {
	wc := mc.GetWitesandCataloger()
	queryRemoteOsm := func(remoteOsmIP string) (witesand.RemotePods, error) {
		log.Info().Msgf("[queryRemoteOsm] querying osm:%s", remoteOsmIP)
		dest := fmt.Sprintf("%s:%s", remoteOsmIP, witesand.HttpServerPort)
		url := fmt.Sprintf("http://%s/localgatewaypods", dest)
		client := &http.Client{}
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set(witesand.HttpRemoteAddrHeader, wc.GetMyIP())
		req.Header.Set(witesand.HttpRemoteClusterIdHeader, wc.GetClusterId())
		resp, err := client.Do(req)
		var remotePods witesand.RemotePods
		if err == nil {
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				err = json.Unmarshal(b, &remotePods)
				if err == nil {
					return remotePods, nil
				}
			}
		}
		log.Info().Msgf("[queryRemoteOsm] err:%+v", err)
		return remotePods, err
	}

	ticker := time.NewTicker(15 * time.Second)
	// run forever
	for {
		<-ticker.C
		for remoteK8sName, remoteK8s := range wc.ListRemoteK8s() {
			remotePods, err := queryRemoteOsm(remoteK8s.OsmIP)
			if err == nil {
				wc.UpdateRemotePods(remoteK8sName, &remotePods)
			}
		}
	}
}

func (mc *MeshCatalog) GetMyIP() string {
        // Get preferred outbound ip of this machine
	myIP := mc.GetWitesandCataloger().GetMyIP()
	if myIP != "" {
		return myIP
	}
	conn, err := net.Dial("udp", witesand.HttpServerPort)
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	myIP = localAddr.IP.String()

	mc.GetWitesandCataloger().RegisterMyIP(myIP)

	return myIP
}

func (mc *MeshCatalog) GetLocalGatewayPods(w http.ResponseWriter, r *http.Request) {
	// learn remote OSM clusterID and address
	remoteAddress := r.Header.Get(witesand.HttpRemoteAddrHeader)
	remoteClusterId := r.Header.Get(witesand.HttpRemoteClusterIdHeader)

	log.Info().Msgf("[GetLocalGatewayPods] remote IP:%s clusterId:%s", remoteAddress, remoteClusterId)

	mc.GetWitesandCataloger().UpdateRemoteK8s(remoteAddress, remoteClusterId)

	list, err := mc.GetWitesandCataloger().ListLocalGatewaypods()
	if err != nil {
		log.Error().Msgf("err fetching local gateway pod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching local gateway pod %+v", err)
	}
}

func (mc *MeshCatalog) GetAllGatewayPods(w http.ResponseWriter, r *http.Request) {
	list, err := mc.GetWitesandCataloger().ListAllGatewaypods()
	if err != nil {
		log.Error().Msgf("err fetching gateway pod %+v", err)
	}

	if err := json.NewEncoder(w).Encode(list); err != nil {
		log.Error().Msgf("err fetching gateway pod %+v", err)
	}
}

func (mc *MeshCatalog) LocalEndpoints(w http.ResponseWriter, r *http.Request) {
	endpointMap, err := mc.ListLocalClusterEndpoints()
	if err != nil {
		log.Error().Msgf("err fetching endpoints %+v", err)
	}

	if err := json.NewEncoder(w).Encode(endpointMap); err != nil {
		log.Error().Msgf("err encoding endpoints %+v", err)
	}
}

func (mc *MeshCatalog) ApigroupMapping(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" || r.Method == "DELETE" || r.Method == "PUT" {
		mc.witesandCatalog.UpdateApigroupMap(w, r.Method, r)
	} else {
		http.Error(w, "Invalid request method.", 405)
		return
	}
}