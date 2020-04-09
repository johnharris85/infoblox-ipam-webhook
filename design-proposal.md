CAPV Infoblox (or other IPAM) Webhook
=====================================

This design proposal is for a webhook capable of dynamically allocating IP addresses to
VSphereMachines from an external IPAM solution (in this case, Infoblox, although
the solution is designed to be generic). CAPV currently allows static IP addresses to
be populated through the IPAddr field but this is of limited value. Many environments are
also unable to use DHCP for a variety of reasons.

## Requirements

Any design needs to satisfy the following requirements:

1. No change to the CAPV types or API.

2. Must allow multiple IPAM providers.

3. No additional constraints around existing CAPV functionality / types.

## Design

The `IPAddr` field is a slice of strings which allows us to accept arbitrary values.
To request an IP address from an external IPAM provider dynamically, the user would
insert a value like `infoblox:<IP_POOL>`.

- Connection details are held in a ConfigMap in-cluster
- Credentials are held in a Secret in-cluster
