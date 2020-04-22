package pkg

import (
	"github.com/johnharris85/infoblox-ipam-webhook/pkg/mocks"
	. "github.com/onsi/ginkgo"

	//"github.com/davecgh/go-spew/spew"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	"testing"
)

func TestManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Manager Suite")
}

var _ = Describe("Infoblox tests", func() {
	var objManager *mocks.ObjectManager
	w := Webhook{
		Client:             nil,
		InfobloxConfigMap:  nil,
		InfobloxAnnotation: "a",
		InfobloxPrefix:     "",
		decoder:            nil,
	}
	BeforeEach(func() {
		objManager = mocks.NewObjectManager()
	})

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
							IPAddrs:     []string{"infoblox:somenetview:somecidr", "notus", "infoblox:somenetview:somecidr"},
						},
						}},
					NumCPUs:   3,
					MemoryMiB: 40,
					DiskGiB:   40,
				},
			},
		}
		_ = w.populateIPs(&vm, objManager)
		Expect(vm.Spec.Network.Devices[0].IPAddrs[0]).To(Equal("0.0.0.0/32"))
		Expect(vm.Spec.Network.Devices[0].IPAddrs[2]).To(Equal("0.0.0.0/32"))
		Expect(vm.Annotations["a"]).To(Equal("infoblox:somenetview:somecidr,infoblox:somenetview:somecidr"))
		//spew.Dump(vm)
	})

	It("release IPs and removes annotations", func() {
		var vm = v1alpha3.VSphereMachine{
			ObjectMeta: v1.ObjectMeta{
				Name:        "vmname",
				Namespace:   "vmnamespace",
				Annotations: map[string]string{
					"a": "infoblox:somenetview:somecidr,infoblox:somenetview:somecidr",
					"b": "test",
				},
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
							IPAddrs:     []string{""},
						},
						}},
					NumCPUs:   3,
					MemoryMiB: 40,
					DiskGiB:   40,
				},
			},
		}
		_ = w.cleanupIPs(&vm, objManager)
		Expect(vm.Annotations).Should(HaveKey("a"))
		Expect(vm.Annotations).ShouldNot(HaveKey("a"))
	})
})
