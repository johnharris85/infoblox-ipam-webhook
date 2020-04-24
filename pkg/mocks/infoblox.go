package mocks

import (
	ib "github.com/infobloxopen/infoblox-go-client"
)

type ObjectManager struct{}

func NewObjectManager() *ObjectManager {
	objMgr := new(ObjectManager)

	return objMgr
}

func (objMgr *ObjectManager) GetARecordByRef(ref string) (*ib.RecordA, error) {
	return &ib.RecordA{
		IBBase:   ib.IBBase{},
		Ref:      ref,
		Ipv4Addr: "0.0.0.0",
		Name:     "",
		View:     "",
		Zone:     "",
		Ea:       nil,
	}, nil
}

func (objMgr *ObjectManager) CreateARecord(netview string, dnsview string, recordname string, cidr string, ipAddr string, ea ib.EA) (*ib.RecordA, error) {
	return &ib.RecordA{
		View:     dnsview,
		Name:     recordname,
		Ea:       ea,
		Ref:      netview,
		Ipv4Addr: "0.0.0.0"}, nil
}

func (objMgr *ObjectManager) DeleteARecord(ref string) (string, error) {
	return "", nil
}
