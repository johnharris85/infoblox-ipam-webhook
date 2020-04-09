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

	infoblox "github.com/infobloxopen/infoblox-go-client"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

// +kubebuilder:webhook:path=/infoblox-ipam,mutating=true,failurePolicy=fail,groups="infrastructure.cluster.x-k8s.io",resources=vspheremachines,verbs=create;delete,versions=v1alpha3,name=mutating.infoblox.ipam.vspheremachines.infrastructure.cluster.x-k8s.io

type Webhook struct {
	Client                     client.Client
	InfobloxSecretName         string
	InfobloxSecretNamespace    string
	InfobloxConfigMapNamespace string
	InfobloxConfigMap          string
	InfobloxAnnotation         string
	decoder                    *admission.Decoder
}

// Handle
func (w *Webhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	vm := &v1alpha3.VSphereMachine{}

	err := w.decoder.Decode(req, vm)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	marshaledVM, err := json.Marshal(vm)
	doWeCare := false

	for _, device := range vm.Spec.VirtualMachineCloneSpec.Network.Devices {
		for _, ip := range device.IPAddrs {
			if strings.HasPrefix(ip, "infoblox") {
				doWeCare = true
			}
		}
	}

	if !doWeCare {
		return admission.PatchResponseFromRaw(req.Object.Raw, marshaledVM)
	}

	//if _, ok := vm.Annotations[w.InfobloxAnnotation]; !ok {
	//	marshaledVM, err := json.Marshal(vm)
	//	if err != nil {
	//		return admission.Errored(http.StatusInternalServerError, err)
	//	}
	//	admission.PatchResponseFromRaw(req.Object.Raw, marshaledVM)
	//}

	infobloxSecret := &corev1.Secret{}
	secret := types.NamespacedName{
		Namespace: w.InfobloxSecretNamespace,
		Name:      w.InfobloxSecretName,
	}
	err = w.Client.Get(ctx, secret, infobloxSecret)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

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
	//cmpType = cloud management platform
	objMgr := infoblox.NewObjectManager(conn, infobloxConfig.Data["cmpType"], infobloxConfig.Data["tenantID"])

	"infoblox:<netview>:<cidr>"

	if req.Operation == admissionv1beta1.Create {
		addr, err := objMgr.AllocateIP(
			"<comes_from_field?>",
			"<comes_from_field?>",
			"",
			"",
			"<comes_from_spec?>",
			"",
		)
		if err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		fmt.Println(addr.IPAddress)
	}

	if req.Operation == admissionv1beta1.Delete {

	}

	if vm.Annotations == nil {
		vm.Annotations = map[string]string{}
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

func setupInfobloxConnector(config corev1.ConfigMap, secret corev1.Secret) (*infoblox.Connector, error) {
	hostConfig := infoblox.HostConfig{
		Host:     config.Data["host"],
		Version:  config.Data["version"],
		Port:     config.Data["port"],
		Username: string(secret.Data["username"]),
		Password: string(secret.Data["password"]),
	}
	transportConfig := infoblox.NewTransportConfig("true", 20, 10)
	requestBuilder := &infoblox.WapiRequestBuilder{}
	requestor := &infoblox.WapiHttpRequestor{}
	return infoblox.NewConnector(hostConfig, transportConfig, requestBuilder, requestor)
}
