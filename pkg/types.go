package pkg

import (
	ib "github.com/infobloxopen/infoblox-go-client"
)

type infobloxObjectManager interface {
	CreateARecord(netview string, dnsview string, recordname string, cidr string, ipAddr string, ea ib.EA) (*ib.RecordA, error)
	GetARecordByRef(ref string) (*ib.RecordA, error)
	DeleteARecord(ref string) (string, error)
}
