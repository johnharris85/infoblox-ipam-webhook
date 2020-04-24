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
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	ib "github.com/infobloxopen/infoblox-go-client"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"net/http"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

// +kubebuilder:webhook:path=/infoblox-ipam,mutating=true,failurePolicy=fail,groups="infrastructure.cluster.x-k8s.io",resources=vspheremachines,verbs=create;delete,versions=v1alpha3,name=mutating.infoblox.ipam.vspheremachines.infrastructure.cluster.x-k8s.io

var logger = logf.Log.WithName("infoblox-webhook")

type InfoBloxIPAM struct {
	InfobloxObjMgr     infobloxObjectManager
	InfobloxAnnotation string
	InfobloxPrefix     string
	decoder            *admission.Decoder
}

// Handle
func (in *InfoBloxIPAM) Handle(_ context.Context, req admission.Request) admission.Response {
	logf.SetLogger(zap.Logger(false))
	log := logger.WithName("handler")
	spew.Dump("req", req)
	vm := &v1alpha3.VSphereMachine{}

	switch req.Operation {
	case admissionv1beta1.Create:
		err := in.decoder.DecodeRaw(req.Object, vm)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		var ipsContainInfobloxPrefix bool
		for _, device := range vm.Spec.VirtualMachineCloneSpec.Network.Devices {
			for _, ip := range device.IPAddrs {
				log.Info(fmt.Sprintf("checking IP: %s", ip))
				if strings.Split(ip, ":")[0] == in.InfobloxPrefix {
					ipsContainInfobloxPrefix = true
				}
			}
		}

		if !ipsContainInfobloxPrefix {
			log.Info("create operation and no IPs with prefix found")
			return admission.Allowed("")
		}
		in.populateIPs(vm)
	case admissionv1beta1.Delete:
		err := in.decoder.DecodeRaw(req.OldObject, vm)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		_, ok := vm.Annotations[in.InfobloxAnnotation]
		if !ok {
			log.Info("delete operation and no annotation found")
			return admission.Allowed("")
		}
		in.cleanupIPs(vm)
	default:
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("unsupported operation: %s", req.Operation))
	}

	marshaledVM, err := json.Marshal(vm)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	var response admission.Response

	switch req.Operation {
	case admissionv1beta1.Create:
		response = admission.PatchResponseFromRaw(req.Object.Raw, marshaledVM)
	case admissionv1beta1.Delete:
		response = admission.Allowed("")
	default:
		return admission.Errored(http.StatusInternalServerError, fmt.Errorf("unsupported operation: %s", req.Operation))
	}

	spew.Dump("response", response)
	return response
}

// InjectDecoder injects the decoder.
func (in *InfoBloxIPAM) InjectDecoder(d *admission.Decoder) error {
	in.decoder = d
	return nil
}

func (in *InfoBloxIPAM) populateIPs(vm *v1alpha3.VSphereMachine) error {
	if vm.Annotations == nil {
		vm.Annotations = map[string]string{in.InfobloxAnnotation: ""}
	}
	var ipAddrAnnotation []string
	for deviceIdx, device := range vm.Spec.VirtualMachineCloneSpec.Network.Devices {
		for ipIdx, ip := range device.IPAddrs {
			if strings.HasPrefix(ip, "infoblox") {
				// Format - "infoblox:<netview>:<dnsview>:<cidr>"
				netview, cidr, dnsview, err := parseInfobloxIPString(ip)
				if err != nil {
					return err //TODO: idempotency? what about if it already exists / doesn't create all first time?
				}
				addr, err := in.InfobloxObjMgr.CreateARecord(netview, dnsview, fmt.Sprintf("%s-%d-%d", vm.Name, deviceIdx, ipIdx), cidr, "", ib.EA{})
				if err != nil {
					return err
				}
				device.IPAddrs[ipIdx] = addr.Ipv4Addr
				ipAddrAnnotation = append(ipAddrAnnotation, addr.Ref)
			}
		}
	}
	vm.Annotations[in.InfobloxAnnotation] = strings.Join(ipAddrAnnotation, ",")
	return nil
}

func (in *InfoBloxIPAM) cleanupIPs(vm *v1alpha3.VSphereMachine) error {
	recordsToRemove := strings.Split(vm.Annotations[in.InfobloxAnnotation], ",")
	for _, recordRef := range recordsToRemove {
		_, err := in.InfobloxObjMgr.DeleteARecord(recordRef)
		if err != nil {
			// TODO: Deal with some records deleted, some not? What error is returned from InfoBlox if a record is
			// already deleted? We can probably ignore that.
			return err
		}
	}
	return nil
}

func parseInfobloxIPString(marker string) (string, string, string, error) {
	splitIP := strings.Split(marker, ":")
	if len(splitIP) != 4 {
		return "", "", "", errors.New("not a valid infoblox string")
	}
	return splitIP[1], splitIP[2], splitIP[3], nil
}
