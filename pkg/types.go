package pkg

import (
	ib "github.com/infobloxopen/infoblox-go-client"
)

type infobloxObjectManager interface {
	AllocateIP(netview string, cidr string, ipAddr string, macAddress string, name string, ea ib.EA) (*ib.FixedAddress, error)
	CreateARecord(netview string, dnsview string, recordname string, cidr string, ipAddr string, ea ib.EA) (*ib.RecordA, error)
	CreateCNAMERecord(canonical string, recordname string, dnsview string, ea ib.EA) (*ib.RecordCNAME, error)
	CreatePTRRecord(netview string, dnsview string, recordname string, cidr string, ipAddr string, ea ib.EA) (*ib.RecordPTR, error)
	DeleteARecord(ref string) (string, error)
	DeleteCNAMERecord(ref string) (string, error)
	DeletePTRRecord(ref string) (string, error)
	ReleaseIP(netview string, cidr string, ipAddr string, macAddr string) (string, error)
}
