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
	"flag"
	ipam "github.com/johnharris85/infoblox-ipam-webhook/pkg"
	"os"
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
	)

	flag.StringVar(
		&infobloxConfigMap,
		"configName",
		"webhook-config",
		"Name of the configmap containing Infoblox connection details.",
	)

	flag.StringVar(
		&infobloxConfigMapNamespace,
		"configNamespace",
		"infoblox",
		"Name of the namespace containing the configmap with Infoblox connection details.",
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

	// Setup a Manager
	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	// Setup webhook
	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	entryLog.Info("registering webhooks to the webhook server")
	hookServer.Register("/infoblox-ipam", &webhook.Admission{Handler: &ipam.Webhook{
		Client:                     mgr.GetClient(),
		InfobloxSecretName:         infobloxSecretName,
		InfobloxSecretNamespace:    infobloxSecretNamespace,
		InfobloxConfigMap:          infobloxConfigMap,
		InfobloxConfigMapNamespace: infobloxConfigMapNamespace,
		InfobloxAnnotation:         infobloxAnnotation,
	}})

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}
