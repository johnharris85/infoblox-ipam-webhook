package mocks

import (
	"fmt"
	ib "github.com/infobloxopen/infoblox-go-client"
)

type ObjectManager struct{}

func NewObjectManager() *ObjectManager {
	objMgr := new(ObjectManager)

	return objMgr
}

func (objMgr *ObjectManager) AllocateIP(netview, cidr, _, _, name string, ea ib.EA) (*ib.FixedAddress, error) {
	return &ib.FixedAddress{
		NetviewName: netview,
		Cidr:        cidr,
		Mac:         ib.MACADDR_ZERO,
		Name:        name,
		Ea:          ea,
		Ref:         fmt.Sprintf("%s%s", netview, name),
		IPAddress:   "0.0.0.0",
	}, nil
}

func (objMgr *ObjectManager) ReleaseIP(_, _, _, _ string) (string, error) {
	return "", nil
}

func (objMgr *ObjectManager) CreateARecord(netview string, dnsview string, recordname string, cidr string, ipAddr string, ea ib.EA) (*ib.RecordA, error) {
	return &ib.RecordA{
		View:     dnsview,
		Name:     recordname,
		Ea:       ea,
		Ref:      fmt.Sprintf("%s%s", netview, recordname),
		Ipv4Addr: "0.0.0.0"}, nil
}

func (objMgr *ObjectManager) DeleteARecord(ref string) (string, error) {
	return "", nil
}

func (objMgr *ObjectManager) CreateCNAMERecord(canonical string, recordname string, dnsview string, ea ib.EA) (*ib.RecordCNAME, error) {
	return &ib.RecordCNAME{
		View:      dnsview,
		Name:      recordname,
		Canonical: canonical,
		Ea:        ea,
		Ref:       fmt.Sprintf("%s%s", canonical, recordname),
	}, nil
}

func (objMgr *ObjectManager) DeleteCNAMERecord(ref string) (string, error) {
	return "", nil
}

func (objMgr *ObjectManager) CreatePTRRecord(netview string, dnsview string, recordname string, cidr string, ipAddr string, ea ib.EA) (*ib.RecordPTR, error) {

	return &ib.RecordPTR{
		View:     dnsview,
		PtrdName: recordname,
		Ea:       ea,
		Ref:      fmt.Sprintf("%s%s", netview, recordname),
		Ipv4Addr: "0.0.0.0/32",
	}, nil
}

func (objMgr *ObjectManager) DeletePTRRecord(ref string) (string, error) {
	return "", nil
}
