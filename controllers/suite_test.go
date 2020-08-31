/*


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

package controllers

import (
	"context"
	monitoringv1alpha1 "github.com/aminasian/prometheus-slo-operator/api/v1alpha1"
	promoperatorv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"log"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
var ctx context.Context

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	ctx = context.Background()

	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	log.Print("finnished starting test environment")

	err = monitoringv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	err = promoperatorv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	log.Print("finnished adding monitoring api to manager scheme")

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	log.Print("finnished starting instantiating k8sclient for testing")

	Context("Finished setting up k8sClient", func() {})

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})

func setupK8sClient() (client.Client, error) {

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	if err != nil {
		return nil, err
	}

	log.Print("finnished starting test environment")

	err = monitoringv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	log.Print("finnished adding monitoring api to manager scheme")

	err = promoperatorv1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}


	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}

	return k8sClient, nil
}

var _ = Describe("TestDeploymentResourceCreation", func() {

	client, err := setupK8sClient()

	Context("After setting up a test environment K8s client", func() {
		It("the returned error should be nil", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})


	deployment, err := newDeploymentResourceForCR(&monitoringv1alpha1.ServiceLevel{
		TypeMeta: v1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:            "test-service-level",
			Namespace:       "default",
			Labels: map[string]string{"app.kubernetes.io/name": "test-service-level"},
		},
		Spec: monitoringv1alpha1.ServiceLevelSpec{
			PrometheusAddress:      "",
			PrometheusName:         "",
			ServiceLevelObjectives: nil,
		},
		Status: monitoringv1alpha1.ServiceLevelStatus{},
	}, "aminasian/prometheus-slo-calculator:0.0.1", true)

	Context("After creating a new K8s Deployment resource struct", func() {
		It("the returned error should be nil", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})


	err = client.Create(context.Background(), deployment)

	Context("After calling the K8s api and creating a new K8s Deployment resource", func() {
		It("the returned error should be nil", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

})

var _ = Describe("TestPodMonitorResourceCreation", func() {
	client, err := setupK8sClient()

	Context("After setting up a test environment K8s client", func() {
		It("the returned error should be nil", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})

	podMonitor, err :=  newPodMonitorForCR(&monitoringv1alpha1.ServiceLevel{
		TypeMeta: v1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:            "test-service-level",
			Namespace:       "default",
			Labels: map[string]string{"app.kubernetes.io/name": "test-service-level"},
		},
		Spec: monitoringv1alpha1.ServiceLevelSpec{
			PrometheusAddress:      "",
			PrometheusName:         "",
			ServiceLevelObjectives: nil,
		},
		Status: monitoringv1alpha1.ServiceLevelStatus{},
	})

	Context("After creating a new K8s PodMonitor resource struct", func() {
		It("the returned error should be nil", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})


	err = client.Create(context.Background(), podMonitor)

	Context("After calling the K8s api and creating a new K8s PodMonitor resource", func() {
		It("the returned error should be nil", func() {
			Expect(err).NotTo(HaveOccurred())
		})
	})
})