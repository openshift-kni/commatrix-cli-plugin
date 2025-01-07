package commatrix

import (
	"bytes"
	"encoding/json"
	"os"

	"github.com/golang/mock/gomock"
	g "github.com/onsi/ginkgo/v2"
	o "github.com/onsi/gomega"
	mock_utils "github.com/openshift-kni/commatrix/pkg/utils/mock"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openshift-kni/commatrix/pkg/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek "k8s.io/client-go/kubernetes/fake"
)

var (
	clientset *client.ClientSet
	mockUtils *mock_utils.MockUtilsInterface
	ctrlTest  *gomock.Controller
)

var (
	testNode = &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
		},
	}

	testPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app-pod",
			Namespace: "test-ns",
			Labels: map[string]string{
				"kubernetes.io/service-name": "test-service",
				"app":                        "test-app",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "test-image:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}

	testService = &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "test-ns",
			Labels: map[string]string{
				"kubernetes.io/service-name": "test-service",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "test-app",
			},
			Ports: []corev1.ServicePort{
				{
					Port:     80,
					Protocol: corev1.ProtocolTCP,
				},
			},
			Type: corev1.ServiceTypeNodePort,
		},
	}

	testEndpointSlice = &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service-endpoints",
			Namespace: "test-ns",
			Labels: map[string]string{
				"kubernetes.io/service-name": "test-service",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "Service",
					Name: "test-service",
				},
			},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints: []discoveryv1.Endpoint{
			{
				NodeName:  &testNode.Name,
				Addresses: []string{"192.168.1.1", "192.168.1.2"},
			},
		},
		Ports: []discoveryv1.EndpointPort{
			{
				Name:     nil,
				Port:     func(i int32) *int32 { return &i }(80),
				Protocol: func(p corev1.Protocol) *corev1.Protocol { return &p }(corev1.ProtocolTCP),
			},
		},
	}
)

var _ = g.Describe("Commatrix Tests", func() {
	g.Context("Create utils and client Matrix", func() {
		g.BeforeEach(func() {
			sch := runtime.NewScheme()

			err := corev1.AddToScheme(sch)
			o.Expect(err).NotTo(o.HaveOccurred())
			err = discoveryv1.AddToScheme(sch)
			o.Expect(err).NotTo(o.HaveOccurred())

			fakeClient := fake.NewClientBuilder().WithScheme(sch).WithObjects(testNode, testPod, testService, testEndpointSlice).Build()
			fakeClientset := fakek.NewSimpleClientset()

			clientset = &client.ClientSet{
				Client:          fakeClient,
				CoreV1Interface: fakeClientset.CoreV1(),
			}

			ctrlTest = gomock.NewController(g.GinkgoT())
			mockUtils = mock_utils.NewMockUtilsInterface(ctrlTest)
			mockUtils.EXPECT().IsBMInfra().Return(false, nil)
			mockUtils.EXPECT().IsSNOCluster().Return(false, nil)
		})

		g.AfterEach(func() {
			ctrlTest.Finish()
		})

		g.It("should create with csv format ", func() {
			var outputBuffer bytes.Buffer
			originalStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			options := &CommatrixOptions{
				format:              "csv",
				customEntriesPath:   "",
				customEntriesFormat: "",
				debug:               false,
				cs:                  clientset,
				utilsHelpers:        mockUtils,
			}
			err := options.Run()
			o.Expect(err).NotTo(o.HaveOccurred())
			w.Close()
			os.Stdout = originalStdout
			_, err = outputBuffer.ReadFrom(r)
			o.Expect(err).NotTo(o.HaveOccurred())

			expectedHeader := "Direction,Protocol,Port,Namespace,Service,Pod,Container,Node Role,Optional\n"
			actualOutput := outputBuffer.String()
			o.Expect(actualOutput).To(o.HavePrefix(expectedHeader))

		})

		g.It("should create with json format", func() {
			var outputBuffer bytes.Buffer
			originalStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			options := &CommatrixOptions{
				format:              "json",
				customEntriesPath:   "",
				customEntriesFormat: "",
				debug:               false,
				cs:                  clientset,
				utilsHelpers:        mockUtils,
			}

			err := options.Run()
			o.Expect(err).NotTo(o.HaveOccurred())

			w.Close()
			os.Stdout = originalStdout
			_, err = outputBuffer.ReadFrom(r)
			o.Expect(err).NotTo(o.HaveOccurred())

			actualOutput := outputBuffer.String()

			var jsonOutput interface{}
			err = json.Unmarshal([]byte(actualOutput), &jsonOutput)
			o.Expect(err).NotTo(o.HaveOccurred(), "output is not valid JSON")

		})
	})

})
