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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
  "k8s.io/client-go/kubernetes"
	"github.com/golang/glog"
	"k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func toAdmissionResponse(err error) *v1beta1.AdmissionResponse {
	return &v1beta1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}


func admitPods(clientset *kubernetes.Clientset, ar v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	glog.V(2).Info("admitting pods")
	podResource := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if ar.Request.Resource != podResource {
		err := fmt.Errorf("expect resource to be %s", podResource)
		glog.Error(err)
		return toAdmissionResponse(err)
	}

	raw := ar.Request.Object.Raw
	pod := corev1.Pod{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(raw, nil, &pod); err != nil {
		glog.Error(err)
		return toAdmissionResponse(err)
	}

	reviewResponse := v1beta1.AdmissionResponse{}
	reviewResponse.Allowed = true

	var msg string
  // Query other pods by using the pod name!
  if v, ok := pod.Labels["job-name"]; ok {

		maxPodCount := 10

		job, jerr := clientset.BatchV1().Jobs(pod.Namespace).Get(v, metav1.GetOptions{})
		if jerr == nil {
			maxPodCount = int(*job.Spec.BackoffLimit)
		}

    selector := fmt.Sprintf("job-name=%s", v)
    pods, err := clientset.CoreV1().Pods(pod.Namespace).List(metav1.ListOptions{
      LabelSelector: selector,
    })

    if err != nil {
      panic(err.Error())
    }
		glog.Info(fmt.Sprintf("There are %d/%d pods for the job %s in the cluster", len(pods.Items), maxPodCount, v))
    if len(pods.Items) >= maxPodCount {
      reviewResponse.Allowed = false
			msg = msg + "Too many pods (over the BackoffLimit) with same job-name! ; "
    }
  }

	if !reviewResponse.Allowed {
		reviewResponse.Result = &metav1.Status{Message: strings.TrimSpace(msg)}
	}
	return &reviewResponse
}



type admitFunc func(*kubernetes.Clientset, v1beta1.AdmissionReview) *v1beta1.AdmissionResponse

func serve(clientset *kubernetes.Clientset, w http.ResponseWriter, r *http.Request, admit admitFunc) {
	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		glog.Errorf("contentType=%s, expect application/json", contentType)
		return
	}

	var reviewResponse *v1beta1.AdmissionResponse
	ar := v1beta1.AdmissionReview{}
	deserializer := codecs.UniversalDeserializer()
	if _, _, err := deserializer.Decode(body, nil, &ar); err != nil {
		glog.Error(err)
		reviewResponse = toAdmissionResponse(err)
	} else {
		reviewResponse = admit(clientset, ar)
	}

	response := v1beta1.AdmissionReview{}
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = ar.Request.UID
	}
	// reset the Object and OldObject, they are not needed in a response.
	ar.Request.Object = runtime.RawExtension{}
	ar.Request.OldObject = runtime.RawExtension{}

	resp, err := json.Marshal(response)
	if err != nil {
		glog.Error(err)
	}
	if _, err := w.Write(resp); err != nil {
		glog.Error(err)
	}
}


func servePods(clientset *kubernetes.Clientset) func (w http.ResponseWriter, r *http.Request) {
  return func (w http.ResponseWriter, r *http.Request) {
    glog.Info("Admission request received")
	   serve(clientset, w, r, admitPods)
  }
}

func main() {

	flag.Parse()

  // Kubernetes client
  clientset := getClient()

	http.HandleFunc("/", servePods(clientset))
  fmt.Printf("Listen at :443")



	server := &http.Server{
		Addr:      ":443",
		TLSConfig: configTLS(clientset),
	}
	// Register itself to k8s api
	go selfRegistration(clientset, serverCert)

	server.ListenAndServeTLS("", "")

}
