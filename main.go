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

package main

import (
	"context"
	"flag"
	ib "github.com/infobloxopen/infoblox-go-client"
	ipam "github.com/johnharris85/infoblox-ipam-webhook/pkg"
	"github.com/johnharris85/infoblox-ipam-webhook/pkg/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"os"
	//"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var log = logf.Log.WithName("infoblox-capv-ipam")

func main() {
	logf.SetLogger(zap.Logger(false))
	entryLog := log.WithName("entrypoint")

	var (
		infobloxSecretName         string
		infobloxSecretNamespace    string
		infobloxConfigMap          string
		infobloxConfigMapNamespace string
		infobloxAnnotation         string
		infobloxPrefix             string
	)

	flag.StringVar(
		&infobloxConfigMap,
		"configName",
		"webhook-config",
		"Name of the configmap containing Infoblox connection / config details.",
	)

	flag.StringVar(
		&infobloxConfigMapNamespace,
		"configNamespace",
		"infoblox",
		"Name of the namespace containing the configmap with Infoblox connection / config details.",
	)

	flag.StringVar(
		&infobloxSecretName,
		"secretName",
		"webhook-credentials",
		"Name of the secret containing Infoblox credentials.",
	)

	flag.StringVar(
		&infobloxSecretNamespace,
		"secretNamespace",
		"infoblox",
		"Name of the namespace containing the secret with Infoblox credentials.",
	)

	flag.StringVar(
		&infobloxAnnotation,
		"annotationName",
		"infoblox.ipam.capv",
		"Name of the annotation containing Infoblox allocation information.",
	)

	flag.StringVar(
		&infobloxPrefix,
		"prefix",
		"infoblox",
		"Name of the prefix to look for in the IPAddr field.",
	)

	// Setup a Manager
	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{Port: 7443})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	// Setup webhook
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	k8sClient := mgr.GetAPIReader()

	sec := corev1.Secret{}
	secret := types.NamespacedName{
		Namespace: infobloxSecretNamespace,
		Name:      infobloxSecretName,
	}
	err = k8sClient.Get(context.Background(), secret, &sec)
	if err != nil {
		entryLog.Error(err, "unable to retrieve infoblox secret")
		os.Exit(1)
	}

	cm := corev1.ConfigMap{}
	config := types.NamespacedName{
		Namespace: infobloxConfigMapNamespace,
		Name:      infobloxConfigMap,
	}
	err = k8sClient.Get(context.Background(), config, &cm)
	if err != nil {
		entryLog.Error(err, "unable to retrieve infoblox configmap")
		os.Exit(1)
	}

	conn, err := setupInfobloxConnector(cm, sec)
	defer conn.Logout()
	if err != nil {
		entryLog.Error(err, "unable to setup infoblox connector")
		os.Exit(1)
	}

	objMgr := ib.NewObjectManager(conn, cm.Data["cmpType"], cm.Data["tenantID"])

	//objMgr := mocks.NewObjectManager()

	//err = v1alpha3.AddToScheme(mgr.GetScheme())

	entryLog.Info("registering webhooks to the webhook server")
	hookServer.Register("/infoblox-ipam", &webhook.Admission{Handler: &ipam.InfoBloxIPAM{
		InfobloxObjMgr:     objMgr,
		InfobloxAnnotation: infobloxAnnotation,
		InfobloxPrefix:     infobloxPrefix,
	}})

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
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
