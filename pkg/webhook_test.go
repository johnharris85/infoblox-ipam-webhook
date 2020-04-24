package pkg

import (
	"context"
	"github.com/johnharris85/infoblox-ipam-webhook/pkg/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

func TestManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Manager Suite")
}

var _ = Describe("Handler tests", func() {

	d, _ := admission.NewDecoder(runtime.NewScheme())

	objManager := mocks.NewObjectManager()
	w := InfoBloxIPAM{
		InfobloxObjMgr:     objManager,
		InfobloxAnnotation: "infoblox",
		InfobloxPrefix:     "infoblox",
		decoder:            d,
	}

	It("mutates", func() {
		request := admission.Request{
			AdmissionRequest: admissionv1beta1.AdmissionRequest{
				Operation: admissionv1beta1.Create,
				Object: runtime.RawExtension{Raw: []byte(`{
  "spec": {
    "datacenter": "dc1",
    "network": {
      "devices": [
        {
          "networkName": "network_test",
          "dhcp4": false,
          "dhcp6": false,
          "ipAddrs": [
            "infoblox:somenetview:somednsview:somecidr",
            "notinfoblox",
            "infoblox:somenetview:somednsview:somecidr"
          ]
        }
      ]
    },
    "numCPUs": 3,
    "memoryMiB": 40,
    "diskGiB": 40,
    "template": "t1"
  }
}`)},
			},
		}
		response := w.Handle(context.Background(), request)
		Expect(len(response.Patches) != 0)
	})

})

var _ = Describe("Method tests", func() {
	objManager := mocks.NewObjectManager()
	w := InfoBloxIPAM{
		InfobloxObjMgr:     objManager,
		InfobloxAnnotation: "a",
		InfobloxPrefix:     "infoblox",
		decoder:            nil,
	}

	It("populates IPs and annotations", func() {
		var vm = v1alpha3.VSphereMachine{
			ObjectMeta: v1.ObjectMeta{
				Name:        "vmname",
				Namespace:   "vmnamespace",
				Annotations: nil,
			},
			Spec: v1alpha3.VSphereMachineSpec{
				VirtualMachineCloneSpec: v1alpha3.VirtualMachineCloneSpec{
					Template:   "t1",
					Datacenter: "dc1",
					Network: v1alpha3.NetworkSpec{
						Devices: []v1alpha3.NetworkDeviceSpec{{
							NetworkName: "network_test",
							DHCP4:       false,
							DHCP6:       false,
							IPAddrs:     []string{"infoblox:somenetview:somednsview:somecidr/23:test.com", "notus", "infoblox:somenetview:somednsview:somecidr/23:test.com"},
						},
						}},
					NumCPUs:   3,
					MemoryMiB: 40,
					DiskGiB:   40,
				},
			},
		}
		_ = w.populateIPs(&vm)
		Expect(vm.Spec.Network.Devices[0].IPAddrs[0]).To(Equal("0.0.0.0/23"))
		Expect(vm.Spec.Network.Devices[0].IPAddrs[2]).To(Equal("0.0.0.0/23"))
		Expect(vm.Annotations["a"]).To(Equal("somenetview,somenetview"))
	})
})
