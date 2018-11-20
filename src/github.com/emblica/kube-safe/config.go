/*
Copyright 2017 The Kubernetes Authors.

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
	"crypto/tls"

	"path/filepath"
	"flag"
	"os"
	"time"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"github.com/golang/glog"
)

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}


// Get a clientset with in-cluster config.
func getClient() *kubernetes.Clientset {

	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	var config *rest.Config
	var err error
	// use the current context in kubeconfig
	config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		glog.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}
	return clientset
}



func configTLS(clientset *kubernetes.Clientset) *tls.Config {

	sCert, err := tls.X509KeyPair(serverCert, serverKey)
	if err != nil {
		glog.Fatal(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{sCert},
		//ClientCAs:    apiserverCA,
		//ClientAuth:   tls.RequireAndVerifyClientCert,
	}
}

// register this example webhook admission controller with the kube-apiserver
// by creating externalAdmissionHookConfigurations.
func selfRegistration(clientset *kubernetes.Clientset, caCert []byte) {
	glog.Info("Registering kube-safe-maxpodcount in 10 seconds...")
	time.Sleep(10 * time.Second)
	client := clientset.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations()
	_, err := client.Get("kube-safe-maxpodcount", metav1.GetOptions{})
	if err == nil {
		if err2 := client.Delete("kube-safe-maxpodcount", nil); err2 != nil {
			glog.Fatal(err2)
		}
	}

	webhookConfig := &v1beta1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "kube-safe-maxpodcount",
		},
		Webhooks: []v1beta1.Webhook{
			{
				Name: "max-podcount.emblica.com",
				Rules: []v1beta1.RuleWithOperations{{
					Operations: []v1beta1.OperationType{v1beta1.Create},
					Rule: v1beta1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
					},
				}},

				ClientConfig: v1beta1.WebhookClientConfig{
					Service: &v1beta1.ServiceReference{
						Namespace: "kube-system",
						Name:      "kube-safe-maxpodcount",
					},
					CABundle: caCert,
				},
			},
		},
	}
	if _, err := client.Create(webhookConfig); err != nil {
		glog.Fatal(err)
	}
	glog.Info("Registered!")
}
