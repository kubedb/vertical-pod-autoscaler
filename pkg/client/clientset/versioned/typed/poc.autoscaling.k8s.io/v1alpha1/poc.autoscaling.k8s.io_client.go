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

package v1alpha1

import (
	serializer "k8s.io/apimachinery/pkg/runtime/serializer"
	v1alpha1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/poc.autoscaling.k8s.io/v1alpha1"
	"k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type PocV1alpha1Interface interface {
	RESTClient() rest.Interface
	VerticalPodAutoscalersGetter
	VerticalPodAutoscalerCheckpointsGetter
}

// PocV1alpha1Client is used to interact with features provided by the poc.autoscaling.k8s.io group.
type PocV1alpha1Client struct {
	restClient rest.Interface
}

func (c *PocV1alpha1Client) VerticalPodAutoscalers(namespace string) VerticalPodAutoscalerInterface {
	return newVerticalPodAutoscalers(c, namespace)
}

func (c *PocV1alpha1Client) VerticalPodAutoscalerCheckpoints(namespace string) VerticalPodAutoscalerCheckpointInterface {
	return newVerticalPodAutoscalerCheckpoints(c, namespace)
}

// NewForConfig creates a new PocV1alpha1Client for the given config.
func NewForConfig(c *rest.Config) (*PocV1alpha1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &PocV1alpha1Client{client}, nil
}

// NewForConfigOrDie creates a new PocV1alpha1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *PocV1alpha1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new PocV1alpha1Client for the given RESTClient.
func New(c rest.Interface) *PocV1alpha1Client {
	return &PocV1alpha1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1alpha1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *PocV1alpha1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
