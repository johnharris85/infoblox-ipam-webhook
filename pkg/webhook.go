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

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	ib "github.com/infobloxopen/infoblox-go-client"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

// +kubebuilder:webhook:path=/infoblox-ipam,mutating=true,failurePolicy=fail,groups="infrastructure.cluster.x-k8s.io",resources=vspheremachines,verbs=create;delete,versions=v1alpha3,name=mutating.infoblox.ipam.vspheremachines.infrastructure.cluster.x-k8s.io

var log = logf.Log.WithName("infoblox-webhook")

type Webhook struct {
	Client             client.Client
	InfobloxConnector  *ib.Connector
	InfobloxConfigMap  *corev1.ConfigMap
	InfobloxAnnotation string
	InfobloxPrefix     string
	decoder            *admission.Decoder
}

// Handle
func (w *Webhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	vm := &v1alpha3.VSphereMachine{}
	err := w.decoder.Decode(req, vm)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.V(5).Info(fmt.Sprintf("captured %s request for %s", req.Operation, vm.Name))

	// Escape as soon as we can if we shouldn't handle the delete request
	_, ok := vm.Annotations[w.InfobloxAnnotation]
	if req.Operation == admissionv1beta1.Delete && !ok {
		// marshaledVM, err := json.Marshal(vm)
		//if err != nil {
		//	return admission.Errored(http.StatusBadRequest, err)
		//}
		log.V(2).Info("delete operation and no annotation found")
		return admission.Allowed("")
	}

	// Escape as soon as we can if we shouldn't handle the create request.
	var ipsContainInfobloxPrefix bool
	for _, device := range vm.Spec.VirtualMachineCloneSpec.Network.Devices {
		for _, ip := range device.IPAddrs {
			log.V(4).Info(fmt.Sprintf("checking IP: %s", ip))
			if strings.Split(ip, ":")[0] == w.InfobloxPrefix {
				ipsContainInfobloxPrefix = true
			}
		}
	}

	if !ipsContainInfobloxPrefix {
		log.V(2).Info("create operation and no IPs with prefix found")
		return admission.Allowed("")
	}

	objMgr := ib.NewObjectManager(w.InfobloxConnector, w.InfobloxConfigMap.Data["cmpType"], w.InfobloxConfigMap.Data["tenantID"])

	if req.Operation == admissionv1beta1.Create {
		w.populateIPs(vm, objMgr)
		log.V(5).Info(fmt.Sprintf("after create - %v", vm))
	}

	if req.Operation == admissionv1beta1.Delete {
		w.cleanupIPs(vm, objMgr)
		log.V(5).Info(fmt.Sprintf("after delete - %v", vm))
	}

	marshaledVM, err := json.Marshal(vm)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledVM)
}

// InjectDecoder injects the decoder.
func (w *Webhook) InjectDecoder(d *admission.Decoder) error {
	w.decoder = d
	return nil
}

func (w *Webhook) populateIPs(vm *v1alpha3.VSphereMachine, objMgr infobloxObjectManager) error {
	if vm.Annotations == nil {
		vm.Annotations = map[string]string{w.InfobloxAnnotation: ""}
	}
	var ipAddrAnnotation []string
	for deviceIdx, device := range vm.Spec.VirtualMachineCloneSpec.Network.Devices {
		for ipIdx, ip := range device.IPAddrs {
			if strings.HasPrefix(ip, "infoblox") {
				// Format - "infoblox:<netview>:<cidr>" TODO: Do we need DNSView?
				splitIP := strings.Split(ip, ":")
				addr, err := objMgr.CreateARecord(splitIP[1], "", fmt.Sprintf("%s-%d-%d", vm.Name, deviceIdx, ipIdx), splitIP[2], "", ib.EA{})
				//addr, err := objMgr.AllocateIP(
				//	splitIP[1],
				//	splitIP[2],
				//	"",
				//	"",
				//	fmt.Sprintf("%s-%d-%d", vm.Name, deviceIdx, ipIdx),
				//	ib.EA{},
				//)
				if err != nil {
					return err
				}
				device.IPAddrs[ipIdx] = addr.Ipv4Addr
				ipAddrAnnotation = append(ipAddrAnnotation, addr.Ref)
			}
		}
	}
	vm.Annotations[w.InfobloxAnnotation] = strings.Join(ipAddrAnnotation, ",")
	return nil
}

func (w *Webhook) cleanupIPs(vm *v1alpha3.VSphereMachine, objMgr infobloxObjectManager) error {
	ipAddrsToRemove := strings.Split(vm.Annotations[w.InfobloxAnnotation], ",")
	for _, ip := range ipAddrsToRemove {
		_, err := objMgr.DeleteARecord(ip)
		if err != nil {
			// TODO: Deal with some records deleted, some not? What error is returned from InfoBlox if a record is
			// already deleted? We can probably ignore that.
			return err
		}
	}
	delete(vm.Annotations, w.InfobloxAnnotation)
	return nil
}
