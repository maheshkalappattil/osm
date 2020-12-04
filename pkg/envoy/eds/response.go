package eds

import (
	xds_discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"github.com/openservicemesh/osm/pkg/catalog"
	"github.com/openservicemesh/osm/pkg/configurator"
	"github.com/openservicemesh/osm/pkg/endpoint"
	"github.com/openservicemesh/osm/pkg/envoy"
	"github.com/openservicemesh/osm/pkg/envoy/cla"
	"github.com/openservicemesh/osm/pkg/service"
)

// NewResponse creates a new Endpoint Discovery Response.
func NewResponse(catalog catalog.MeshCataloger, proxy *envoy.Proxy, _ *xds_discovery.DiscoveryRequest, _ configurator.Configurator) (*xds_discovery.DiscoveryResponse, error) {
	svcList, err := catalog.GetServicesFromEnvoyCertificate(proxy.GetCommonName())
	if err != nil {
		log.Error().Err(err).Msgf("Error looking up MeshService for Envoy with CN=%q", proxy.GetCommonName())
		return nil, err
	}
	// Github Issue #1575
	proxyServiceName := svcList[0]

<<<<<<< HEAD
	allTrafficPolicies, err := catalog.ListTrafficPolicies(proxyServiceName)
	log.Debug().Msgf("EDS svc %s allTrafficPolicies %+v", proxyServiceName, allTrafficPolicies)

=======
	outboundServices, err := catalog.ListAllowedOutboundServices(proxyServiceName)
>>>>>>> c614ca2db542271efd6f7b2b106b9d046dc64b90
	if err != nil {
		log.Error().Err(err).Msgf("Error listing outbound services for proxy %q", proxyServiceName)
		return nil, err
	}

<<<<<<< HEAD
	allServicesEndpoints := make(map[service.MeshServicePort][]endpoint.Endpoint)
	for _, trafficPolicy := range allTrafficPolicies {
		isSourceService := trafficPolicy.Source.Equals(proxyServiceName)
		if isSourceService {
			destService := trafficPolicy.Destination.GetMeshService()
			serviceEndpoints, err := catalog.ListEndpointsForService(destService)
			if err != nil {
				log.Error().Err(err).Msgf("Failed listing endpoints for proxy %s", proxyServiceName)
				return nil, err
			}
			destServicePort := trafficPolicy.Destination
			if destServicePort.Port == 0  {
				allServicesEndpoints[destServicePort] = serviceEndpoints
				continue
			}
			// if port specified, filter based on port
			filteredEndpoints := make([]endpoint.Endpoint, 0)
			for _, endpoint := range serviceEndpoints {
				if int(endpoint.Port) != destServicePort.Port {
					continue
				}
				filteredEndpoints = append(filteredEndpoints, endpoint)
			}
			allServicesEndpoints[destServicePort] = filteredEndpoints
=======
	outboundServicesEndpoints := make(map[service.MeshService][]endpoint.Endpoint)
	for _, dstSvc := range outboundServices {
		endpoints, err := catalog.ListEndpointsForService(dstSvc)
		if err != nil {
			log.Error().Err(err).Msgf("Failed listing endpoints for service %s", dstSvc)
			continue
>>>>>>> c614ca2db542271efd6f7b2b106b9d046dc64b90
		}
		outboundServicesEndpoints[dstSvc] = endpoints
	}

	log.Trace().Msgf("Outbound service endpoints for proxy %s: %v", proxyServiceName, outboundServicesEndpoints)

	var protos []*any.Any
	for svc, endpoints := range outboundServicesEndpoints {
		loadAssignment := cla.NewClusterLoadAssignment(svc, endpoints)
		proto, err := ptypes.MarshalAny(loadAssignment)
		if err != nil {
			log.Error().Err(err).Msgf("Error marshalling EDS payload for proxy %s: %+v", proxyServiceName, loadAssignment)
			continue
		}
		protos = append(protos, proto)
	}

	log.Debug().Msgf("EDS url:%s protos: %+v", string(envoy.TypeEDS), protos)
	resp := &xds_discovery.DiscoveryResponse{
		Resources: protos,
		TypeUrl:   string(envoy.TypeEDS),
	}
	return resp, nil
}
