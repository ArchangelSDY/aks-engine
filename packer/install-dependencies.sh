#!/bin/bash
source /home/packer/provision_installs.sh
source /home/packer/provision_source.sh
source /home/packer/packer_source.sh
source /home/packer/cis.sh

RELEASE_NOTES_FILEPATH=/var/log/azure/golden-image-install.complete

echo "Starting build on " $(date) > ${RELEASE_NOTES_FILEPATH}
echo "Using kernel:" >> ${RELEASE_NOTES_FILEPATH}
tee -a ${RELEASE_NOTES_FILEPATH} < /proc/version

copyPackerFiles

echo ""
echo "Components downloaded in this VHD build (some of the below components might get deleted during cluster provisioning if they are not needed):" >> ${RELEASE_NOTES_FILEPATH}

installDeps
cat << EOF >> ${RELEASE_NOTES_FILEPATH}
  - apt-transport-https
  - auditd
  - blobfuse
  - ca-certificates
  - ceph-common
  - cgroup-lite
  - cifs-utils
  - conntrack
  - cracklib-runtime
  - ebtables
  - ethtool
  - fuse
  - git
  - glusterfs-client
  - init-system-helpers
  - iproute2
  - ipset
  - iptables
  - jq
  - libpam-pwquality
  - libpwquality-tools
  - mount
  - nfs-common
  - pigz socat
  - util-linux
  - xz-utils
  - zip
EOF

if [[ ${UBUNTU_RELEASE} == "18.04" ]]; then
  overrideNetworkConfig
fi

ETCD_VERSION="3.3.13"
ETCD_DOWNLOAD_URL="https://acs-mirror.azureedge.net/github-coreos"
installEtcd
echo "  - etcd v${ETCD_VERSION}" >> ${RELEASE_NOTES_FILEPATH}

MOBY_VERSION="3.0.6"
installMoby
echo "  - moby v${MOBY_VERSION}" >> ${RELEASE_NOTES_FILEPATH}
installGPUDrivers
echo "  - nvidia-docker2 nvidia-container-runtime" >> ${RELEASE_NOTES_FILEPATH}

VNET_CNI_VERSIONS="
1.0.25
1.0.24
"
for VNET_CNI_VERSION in $VNET_CNI_VERSIONS; do
    VNET_CNI_PLUGINS_URL="https://acs-mirror.azureedge.net/cni/azure-vnet-cni-linux-amd64-v${VNET_CNI_VERSION}.tgz"
    downloadAzureCNI
    echo "  - Azure CNI version ${VNET_CNI_VERSION}" >> ${RELEASE_NOTES_FILEPATH}
done

CNI_PLUGIN_VERSIONS="
0.7.5
0.7.1
"
for CNI_PLUGIN_VERSION in $CNI_PLUGIN_VERSIONS; do
    CNI_PLUGINS_URL="https://acs-mirror.azureedge.net/cni/cni-plugins-amd64-v${CNI_PLUGIN_VERSION}.tgz"
    downloadCNI
    echo "  - CNI plugin version ${CNI_PLUGIN_VERSION}" >> ${RELEASE_NOTES_FILEPATH}
done

CONTAINERD_VERSIONS="
1.2.4
1.1.6
1.1.5
"
CONTAINERD_DOWNLOAD_URL_BASE="https://storage.googleapis.com/cri-containerd-release/"
for CONTAINERD_VERSION in ${CONTAINERD_VERSIONS}; do
    downloadContainerd
    echo "  - containerd version ${CONTAINERD_VERSION}" >> ${RELEASE_NOTES_FILEPATH}
done

installImg
echo "  - img" >> ${RELEASE_NOTES_FILEPATH}

echo "Docker images pre-pulled:" >> ${RELEASE_NOTES_FILEPATH}

DASHBOARD_VERSIONS="1.10.1"
for DASHBOARD_VERSION in ${DASHBOARD_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/kubernetes-dashboard-amd64:v${DASHBOARD_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

EXECHEALTHZ_VERSIONS="1.2"
for EXECHEALTHZ_VERSION in ${EXECHEALTHZ_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/exechealthz-amd64:${EXECHEALTHZ_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

ADDON_RESIZER_VERSIONS="
1.8.5
1.8.4
1.8.1
1.7
"
for ADDON_RESIZER_VERSION in ${ADDON_RESIZER_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/addon-resizer:${ADDON_RESIZER_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

HEAPSTER_VERSIONS="
1.5.4
1.5.3
1.5.1
"
for HEAPSTER_VERSION in ${HEAPSTER_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/heapster-amd64:v${HEAPSTER_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

METRICS_SERVER_VERSIONS="0.2.1"
for METRICS_SERVER_VERSION in ${METRICS_SERVER_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/metrics-server-amd64:v${METRICS_SERVER_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

KUBE_DNS_VERSIONS="
1.15.4
1.15.0
1.14.13
1.14.5
"
for KUBE_DNS_VERSION in ${KUBE_DNS_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/k8s-dns-kube-dns-amd64:${KUBE_DNS_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

KUBE_ADDON_MANAGER_VERSIONS="
9.0.2
9.0.1
9.0
8.9.1
8.9
8.8
8.7
8.6
"
for KUBE_ADDON_MANAGER_VERSION in ${KUBE_ADDON_MANAGER_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/kube-addon-manager-amd64:v${KUBE_ADDON_MANAGER_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

KUBE_DNS_MASQ_VERSIONS="
1.15.4
1.15.0
1.14.10
1.14.8
1.14.5
"
for KUBE_DNS_MASQ_VERSION in ${KUBE_DNS_MASQ_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/k8s-dns-dnsmasq-nanny-amd64:${KUBE_DNS_MASQ_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

PAUSE_VERSIONS="3.1"
for PAUSE_VERSION in ${PAUSE_VERSIONS}; do
    # Image 'mcr.microsoft.com/k8s/azurestack/core/pause-amd64' is the same as 'k8s.gcr.io/pause-amd64'
    # At the time, re-tagging and pushing to mcr hub seemed simpler than changing how `defaults-kubelet.go` sets `--pod-infra-container-image`
    for IMAGE_BASE in k8s.gcr.io mcr.microsoft.com/k8s/azurestack/core; do
      CONTAINER_IMAGE="${IMAGE_BASE}/pause-amd64:${PAUSE_VERSION}"
      pullContainerImage "docker" ${CONTAINER_IMAGE}
      echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
    done
done

TILLER_VERSIONS="
2.11.0
2.8.1
"
for TILLER_VERSION in ${TILLER_VERSIONS}; do
    CONTAINER_IMAGE="gcr.io/kubernetes-helm/tiller:v${TILLER_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

CLUSTER_AUTOSCALER_VERSIONS="
1.15.1
1.15.0
1.14.4
1.14.2
1.14.0
1.13.6
1.13.4
1.13.2
1.13.1
1.12.7
1.12.5
1.12.3
1.12.2
1.3.9
1.3.8
1.3.7
1.3.4
1.3.3
1.2.5
1.2.2
"
for CLUSTER_AUTOSCALER_VERSION in ${CLUSTER_AUTOSCALER_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/cluster-autoscaler:v${CLUSTER_AUTOSCALER_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

K8S_DNS_SIDECAR_VERSIONS="
1.14.10
1.14.8
"
for K8S_DNS_SIDECAR_VERSION in ${K8S_DNS_SIDECAR_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/k8s-dns-sidecar-amd64:${K8S_DNS_SIDECAR_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

CORE_DNS_VERSIONS="
1.5.0
1.3.1
1.2.6
1.2.2
"
for CORE_DNS_VERSION in ${CORE_DNS_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/coredns:${CORE_DNS_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

RESCHEDULER_VERSIONS="
0.4.0
0.3.1
"
for RESCHEDULER_VERSION in ${RESCHEDULER_VERSIONS}; do
    CONTAINER_IMAGE="k8s.gcr.io/rescheduler:v${RESCHEDULER_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

VIRTUAL_KUBELET_VERSIONS="latest"
for VIRTUAL_KUBELET_VERSION in ${VIRTUAL_KUBELET_VERSIONS}; do
    CONTAINER_IMAGE="microsoft/virtual-kubelet:${VIRTUAL_KUBELET_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

AZURE_CNIIMAGEBASE="mcr.microsoft.com/containernetworking"
AZURE_CNI_NETWORKMONITOR_VERSIONS="
0.0.6
0.0.5
"
for AZURE_CNI_NETWORKMONITOR_VERSION in ${AZURE_CNI_NETWORKMONITOR_VERSIONS}; do
    CONTAINER_IMAGE="${AZURE_CNIIMAGEBASE}/networkmonitor:v${AZURE_CNI_NETWORKMONITOR_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

AZURE_NPM_VERSIONS="1.0.18"
for AZURE_NPM_VERSION in ${AZURE_NPM_VERSIONS}; do
    CONTAINER_IMAGE="${AZURE_CNIIMAGEBASE}/azure-npm:v${AZURE_NPM_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

NVIDIA_DEVICE_PLUGIN_VERSIONS="
1.11
1.10
"
for NVIDIA_DEVICE_PLUGIN_VERSION in ${NVIDIA_DEVICE_PLUGIN_VERSIONS}; do
    CONTAINER_IMAGE="nvidia/k8s-device-plugin:${NVIDIA_DEVICE_PLUGIN_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

TUNNELFRONT_VERSIONS="v1.9.2-v4.0.4"
for TUNNELFRONT_VERSION in ${TUNNELFRONT_VERSIONS}; do
    CONTAINER_IMAGE="docker.io/deis/hcp-tunnel-front:${TUNNELFRONT_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

KUBE_SVC_REDIRECT_VERSIONS="1.0.2"
for KUBE_SVC_REDIRECT_VERSION in ${KUBE_SVC_REDIRECT_VERSIONS}; do
    CONTAINER_IMAGE="docker.io/deis/kube-svc-redirect:v${KUBE_SVC_REDIRECT_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

KV_FLEXVOLUME_VERSIONS="0.0.7"
for KV_FLEXVOLUME_VERSION in ${KV_FLEXVOLUME_VERSIONS}; do
    CONTAINER_IMAGE="mcr.microsoft.com/k8s/flexvolume/keyvault-flexvolume:v${KV_FLEXVOLUME_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

BLOBFUSE_FLEXVOLUME_VERSIONS="1.0.8"
for BLOBFUSE_FLEXVOLUME_VERSION in ${BLOBFUSE_FLEXVOLUME_VERSIONS}; do
    CONTAINER_IMAGE="mcr.microsoft.com/k8s/flexvolume/blobfuse-flexvolume:${BLOBFUSE_FLEXVOLUME_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

IP_MASQ_AGENT_VERSIONS="
2.3.0
2.0.0
"
for IP_MASQ_AGENT_VERSION in ${IP_MASQ_AGENT_VERSIONS}; do
    # TODO remove the gcr.io/google-containers image once AKS switches to use k8s.gcr.io
    DEPRECATED_CONTAINER_IMAGE="gcr.io/google-containers/ip-masq-agent-amd64:v${IP_MASQ_AGENT_VERSION}"
    pullContainerImage "docker" ${DEPRECATED_CONTAINER_IMAGE}
    echo "  - ${DEPRECATED_CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}

    CONTAINER_IMAGE="k8s.gcr.io/ip-masq-agent-amd64:v${IP_MASQ_AGENT_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

NGINX_VERSIONS="1.13.12-alpine"
for NGINX_VERSION in ${NGINX_VERSIONS}; do
    CONTAINER_IMAGE="nginx:${NGINX_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

KMS_PLUGIN_VERSIONS="0.0.9"
for KMS_PLUGIN_VERSION in ${KMS_PLUGIN_VERSIONS}; do
    CONTAINER_IMAGE="mcr.microsoft.com/k8s/kms/keyvault:v${KMS_PLUGIN_VERSION}"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

FLANNEL_VERSIONS="
0.10.0
0.8.0
"
for FLANNEL_VERSION in ${FLANNEL_VERSIONS}; do
    CONTAINER_IMAGE="quay.io/coreos/flannel:v${FLANNEL_VERSION}-amd64"
    pullContainerImage "docker" ${CONTAINER_IMAGE}
    echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
done

pullContainerImage "docker" "busybox"
echo "  - busybox" >> ${RELEASE_NOTES_FILEPATH}

# TODO: fetch supported k8s versions from an aks-engine command instead of hardcoding them here
K8S_VERSIONS="
1.15.2
1.15.2-azs
1.15.1
1.15.1-azs
1.14.5
1.14.5-azs
1.14.4
1.14.4-azs
1.13.9
1.13.9-azs
1.13.8
1.13.8-azs
1.12.8
1.12.8-azs
1.12.7
1.12.7-azs
1.11.10
1.11.10-azs
1.11.9
1.11.9-azs
1.10.13
1.10.12
"
for KUBERNETES_VERSION in ${K8S_VERSIONS}; do
    if [[ $KUBERNETES_VERSION == *"azs"* ]]; then
      HYPERKUBE_URL="mcr.microsoft.com/k8s/azurestack/core/hyperkube-amd64:v${KUBERNETES_VERSION}"
    else
      HYPERKUBE_URL="k8s.gcr.io/hyperkube-amd64:v${KUBERNETES_VERSION}"
      CONTAINER_IMAGE="k8s.gcr.io/cloud-controller-manager-amd64:v${KUBERNETES_VERSION}"
      pullContainerImage "docker" ${CONTAINER_IMAGE}
      echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}
    fi
    extractHyperkube "docker"
    echo "  - ${HYPERKUBE_URL}" >> ${RELEASE_NOTES_FILEPATH}
done

# TODO: remove once ACR is available on Azure Stack
CONTAINER_IMAGE="registry:2.7.1"
pullContainerImage "docker" ${CONTAINER_IMAGE}
echo "  - ${CONTAINER_IMAGE}" >> ${RELEASE_NOTES_FILEPATH}

df -h

# warn at 75% space taken
[ -s $(df -P | grep '/dev/sda1' | awk '0+$5 >= 75 {print}') ] || echo "WARNING: 75% of /dev/sda1 is used" >> ${RELEASE_NOTES_FILEPATH}
# error at 90% space taken
[ -s $(df -P | grep '/dev/sda1' | awk '0+$5 >= 90 {print}') ] || exit 1

{
  echo "Install completed successfully on " $(date)
  echo "VSTS Build NUMBER: ${BUILD_NUMBER}"
  echo "VSTS Build ID: ${BUILD_ID}"
  echo "Commit: ${COMMIT}"
  echo "Feature flags: ${FEATURE_FLAGS}"
} >> ${RELEASE_NOTES_FILEPATH}

# The below statements are used to extract release notes from the packer output
set +x
echo "START_OF_NOTES"
cat ${RELEASE_NOTES_FILEPATH}
echo "END_OF_NOTES"
set -x

# Move logs from VHD creation out of /var/log
sudo mv /var/log /var/log.vhd
sudo mkdir /var/log

applyCIS
