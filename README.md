## CAPV Infoblox IPAM Webhook

kind create cluster
export VSPHERE_USERNAME=a
export VSPHERE_PASSWORD=a
k create ns infoblox

k create configmap -n infoblox webhook-config --from-literal=host=config1 --from-literal=version=config2 --from-literal=port=anlaf --from-literal=cmpType=afnjaf --from-literal=tenantID=afjnaga

kubectl create secret -n infoblox generic webhook-credentials --from-literal=username=supersecret --from-literal=password=topsecret

clusterctl init --infrastructure vsphere --target-namespace default

docker build -t johnharris85/infoblox-ipam-webhook:master .

k apply-f deploy/deploy.yaml 

k apply -f vspheremachine.yaml