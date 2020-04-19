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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

// +kubebuilder:webhook:path=/infoblox-ipam,mutating=true,failurePolicy=fail,groups="infrastructure.cluster.x-k8s.io",resources=vspheremachines,verbs=create;delete,versions=v1alpha3,name=mutating.infoblox.ipam.vspheremachines.infrastructure.cluster.x-k8s.io

var log = logf.Log.WithName("infoblox-webhook")

type Webhook struct {
	Client                     client.Client
	InfobloxSecretName         string
	InfobloxSecretNamespace    string
	InfobloxConfigMapNamespace string
	InfobloxConfigMap          string
	InfobloxAnnotation         string
	InfobloxPrefix string
	decoder                    *admission.Decoder
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

	// Retrieve the Secret containing username / password for Infoblox.
	infobloxSecret := &corev1.Secret{}
	secret := types.NamespacedName{
		Namespace: w.InfobloxSecretNamespace,
		Name:      w.InfobloxSecretName,
	}
	err = w.Client.Get(ctx, secret, infobloxSecret)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Retrieve the ConfigMap containing configuration for Infoblox.
	infobloxConfig := &corev1.ConfigMap{}
	config := types.NamespacedName{
		Namespace: w.InfobloxConfigMapNamespace,
		Name:      w.InfobloxConfigMap,
	}
	err = w.Client.Get(ctx, config, infobloxConfig)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	conn, err := setupInfobloxConnector(*infobloxConfig, *infobloxSecret)
	defer conn.Logout()
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	objMgr := ib.NewObjectManager(conn, infobloxConfig.Data["cmpType"], infobloxConfig.Data["tenantID"])

	if req.Operation == admissionv1beta1.Create {
		w.populateIPs(vm, objMgr)
		log.V(5).Info(fmt.Sprintf("after create - %s", vm))
	}

	if req.Operation == admissionv1beta1.Delete {
		w.cleanupIPs(vm, objMgr)
		log.V(5).Info(fmt.Sprintf("after delete - %s", vm))
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

func (w *Webhook) populateIPs(vm *v1alpha3.VSphereMachine, objMgr *ib.ObjectManager) error {
	if vm.Annotations == nil {
		vm.Annotations = map[string]string{w.InfobloxAnnotation: ""}
	}
	var ipAddrAnnotation []string
	for deviceIdx, device := range vm.Spec.VirtualMachineCloneSpec.Network.Devices { // check this if weird
		for ipIdx, ip := range device.IPAddrs {
			if strings.HasPrefix(ip, "infoblox") {
				// Format - "infoblox:<netview>:<cidr>"
				splitIP := strings.Split(ip, ":")
				addr, err := objMgr.AllocateIP(
					splitIP[1],
					splitIP[2],
					"",
					"",
					fmt.Sprintf("%s-%d-%d", vm.Name, deviceIdx, ipIdx),
					ib.EA{},
				)
				if err != nil {
					return err
				}
				device.IPAddrs[ipIdx] = addr.IPAddress
				ipAddrAnnotation = append(ipAddrAnnotation, ip)
			}
		}
	}
	vm.Annotations[w.InfobloxAnnotation] = strings.Join(ipAddrAnnotation, ",")
	return nil
}

func (w *Webhook) cleanupIPs(vm *v1alpha3.VSphereMachine, objMgr *ib.ObjectManager) error {
	ipAddrsToRemove := strings.Split(vm.Annotations[w.InfobloxAnnotation], ",")
	for _, ip := range ipAddrsToRemove {
		splitIP := strings.Split(ip, ":")
		_, err := objMgr.ReleaseIP(splitIP[1], splitIP[2], splitIP[0], ib.MACADDR_ZERO)
		if err != nil {
			return err
		}
	}
	delete(vm.Annotations, w.InfobloxAnnotation)
	return nil
}

func setupInfobloxConnector(config corev1.ConfigMap, secret corev1.Secret) (*ib.Connector, error) {
	hostConfig := ib.HostConfig{
		Host:     config.Data["host"],
		Version:  config.Data["version"],
		Port:     config.Data["port"],
		Username: string(secret.Data["username"]),
		Password: string(secret.Data["password"]),
	}
	transportConfig := ib.NewTransportConfig("true", 20, 10)
	requestBuilder := &ib.WapiRequestBuilder{}
	requestor := &ib.WapiHttpRequestor{}
	return ib.NewConnector(hostConfig, transportConfig, requestBuilder, requestor)
}
