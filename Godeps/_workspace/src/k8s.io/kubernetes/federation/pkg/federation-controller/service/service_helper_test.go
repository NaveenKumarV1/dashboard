/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package service

import (
	"reflect"
	"testing"

	"k8s.io/kubernetes/pkg/api"
)

func buildServiceStatus(ingresses [][]string) api.LoadBalancerStatus {
	status := api.LoadBalancerStatus{
		Ingress: []api.LoadBalancerIngress{},
	}
	for _, element := range ingresses {
		ingress := api.LoadBalancerIngress{IP: element[0], Hostname: element[1]}
		status.Ingress = append(status.Ingress, ingress)
	}
	return status
}

func TestProcessServiceUpdate(t *testing.T) {
	cc := clusterClientCache{
		clientMap: make(map[string]*clusterCache),
	}
	tests := []struct {
		name             string
		cachedService    *cachedService
		service          *api.Service
		clusterName      string
		expectNeedUpdate bool
		expectStatus     api.LoadBalancerStatus
	}{
		{
			"no-cache",
			&cachedService{
				lastState:        &api.Service{},
				serviceStatusMap: make(map[string]api.LoadBalancerStatus),
			},
			&api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip1", ""}})}},
			"foo",
			true,
			buildServiceStatus([][]string{{"ip1", ""}}),
		},
		{
			"same-ingress",
			&cachedService{
				lastState: &api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip1", ""}})}},
				serviceStatusMap: map[string]api.LoadBalancerStatus{
					"foo1": {Ingress: []api.LoadBalancerIngress{{"ip1", ""}}},
				},
			},
			&api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip1", ""}})}},
			"foo1",
			false,
			buildServiceStatus([][]string{{"ip1", ""}}),
		},
		{
			"diff-cluster",
			&cachedService{
				lastState: &api.Service{
					ObjectMeta: api.ObjectMeta{Name: "bar1"},
				},
				serviceStatusMap: map[string]api.LoadBalancerStatus{
					"foo2": {Ingress: []api.LoadBalancerIngress{{"ip1", ""}}},
				},
			},
			&api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip1", ""}})}},
			"foo1",
			true,
			buildServiceStatus([][]string{{"ip1", ""}}),
		},
		{
			"diff-ingress",
			&cachedService{
				lastState: &api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip4", ""}, {"ip1", ""}, {"ip2", ""}})}},
				serviceStatusMap: map[string]api.LoadBalancerStatus{
					"foo1": buildServiceStatus([][]string{{"ip4", ""}, {"ip1", ""}, {"ip2", ""}}),
				},
			},
			&api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip2", ""}, {"ip3", ""}, {"ip5", ""}})}},
			"foo1",
			true,
			buildServiceStatus([][]string{{"ip2", ""}, {"ip3", ""}, {"ip5", ""}}),
		},
	}
	for _, test := range tests {
		result := cc.processServiceUpdate(test.cachedService, test.service, test.clusterName)
		if test.expectNeedUpdate != result {
			t.Errorf("Test failed for %s, expected %v, saw %v", test.name, test.expectNeedUpdate, result)
		}
		if !reflect.DeepEqual(test.expectStatus, test.cachedService.lastState.Status.LoadBalancer) {
			t.Errorf("Test failed for %s, expected %v, saw %v", test.name, test.expectStatus, test.cachedService.lastState.Status.LoadBalancer)
		}
	}
}

func TestProcessServiceDeletion(t *testing.T) {
	cc := clusterClientCache{
		clientMap: make(map[string]*clusterCache),
	}
	tests := []struct {
		name             string
		cachedService    *cachedService
		service          *api.Service
		clusterName      string
		expectNeedUpdate bool
		expectStatus     api.LoadBalancerStatus
	}{
		{
			"same-ingress",
			&cachedService{
				lastState: &api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip1", ""}})}},
				serviceStatusMap: map[string]api.LoadBalancerStatus{
					"foo1": {Ingress: []api.LoadBalancerIngress{{"ip1", ""}}},
				},
			},
			&api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip1", ""}})}},
			"foo1",
			true,
			buildServiceStatus([][]string{}),
		},
		{
			"diff-ingress",
			&cachedService{
				lastState: &api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip4", ""}, {"ip1", ""}, {"ip2", ""}, {"ip3", ""}, {"ip5", ""}, {"ip6", ""}, {"ip8", ""}})}},
				serviceStatusMap: map[string]api.LoadBalancerStatus{
					"foo1": buildServiceStatus([][]string{{"ip1", ""}, {"ip2", ""}, {"ip3", ""}}),
					"foo2": buildServiceStatus([][]string{{"ip5", ""}, {"ip6", ""}, {"ip8", ""}}),
				},
			},
			&api.Service{Status: api.ServiceStatus{LoadBalancer: buildServiceStatus([][]string{{"ip1", ""}, {"ip2", ""}, {"ip3", ""}})}},
			"foo1",
			true,
			buildServiceStatus([][]string{{"ip4", ""}, {"ip5", ""}, {"ip6", ""}, {"ip8", ""}}),
		},
	}
	for _, test := range tests {
		result := cc.processServiceDeletion(test.cachedService, test.clusterName)
		if test.expectNeedUpdate != result {
			t.Errorf("Test failed for %s, expected %v, saw %v", test.name, test.expectNeedUpdate, result)
		}
		if !reflect.DeepEqual(test.expectStatus, test.cachedService.lastState.Status.LoadBalancer) {
			t.Errorf("Test failed for %s, expected %+v, saw %+v", test.name, test.expectStatus, test.cachedService.lastState.Status.LoadBalancer)
		}
	}
}
