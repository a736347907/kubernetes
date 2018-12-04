/*
Copyright 2018 The Kubernetes Authors.

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

package cloudresource

import (
	"reflect"
	"testing"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/kubernetes/pkg/cloudprovider/providers/fake"
)

func createNodeInternalIPAddress(address string) []v1.NodeAddress {
	return []v1.NodeAddress{
		{
			Type:    v1.NodeInternalIP,
			Address: address,
		},
	}
}

func TestNodeAddressesRequest(t *testing.T) {
	syncPeriod := 300 * time.Millisecond
	maxRetry := 5
	cloud := &fake.FakeCloud{
		Addresses: createNodeInternalIPAddress("10.0.1.12"),
		// Set the request delay so the manager timeouts and collects the node addresses later
		RequestDelay: 400 * time.Millisecond,
	}
	stopCh := make(chan struct{})
	defer close(stopCh)

	manager := NewSyncManager(cloud, "defaultNode", syncPeriod).(*cloudResourceSyncManager)
	go manager.Run(stopCh)

	nodeAddresses, err := manager.NodeAddresses()
	if err != nil {
		t.Errorf("Unexpected err: %q\n", err)
	}
	if !reflect.DeepEqual(nodeAddresses, cloud.Addresses) {
		t.Errorf("Unexpected diff of node addresses: %v", diff.ObjectReflectDiff(nodeAddresses, cloud.Addresses))
	}

	// Change the IP address
	cloud.SetNodeAddresses(createNodeInternalIPAddress("10.0.1.13"))

	// Wait until the IP address changes
	for i := 0; i < maxRetry; i++ {
		nodeAddresses, err := manager.NodeAddresses()
		t.Logf("nodeAddresses: %#v, err: %v", nodeAddresses, err)
		if err != nil {
			t.Errorf("Unexpected err: %q\n", err)
		}
		// It is safe to read cloud.Addresses since no routine is changing the value at the same time
		if err == nil && nodeAddresses[0].Address != cloud.Addresses[0].Address {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if err != nil {
			t.Errorf("Unexpected err: %q\n", err)
		}
		return
	}
	t.Errorf("Timeout waiting for %q address to appear", cloud.Addresses[0].Address)
}
