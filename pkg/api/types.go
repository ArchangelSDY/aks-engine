// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net"
	neturl "net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	v20170831 "github.com/Azure/aks-engine/pkg/api/agentPoolOnlyApi/v20170831"
	v20180331 "github.com/Azure/aks-engine/pkg/api/agentPoolOnlyApi/v20180331"
	"github.com/Azure/aks-engine/pkg/api/common"
	v20160330 "github.com/Azure/aks-engine/pkg/api/v20160330"
	v20160930 "github.com/Azure/aks-engine/pkg/api/v20160930"
	v20170131 "github.com/Azure/aks-engine/pkg/api/v20170131"
	v20170701 "github.com/Azure/aks-engine/pkg/api/v20170701"
	"github.com/Azure/aks-engine/pkg/api/vlabs"
	"github.com/Azure/aks-engine/pkg/helpers"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/blang/semver"
)

// TypeMeta describes an individual API model object
type TypeMeta struct {
	// APIVersion is on every object
	APIVersion string `json:"apiVersion"`
}

// ResourcePurchasePlan defines resource plan as required by ARM
// for billing purposes.
type ResourcePurchasePlan struct {
	Name          string `json:"name"`
	Product       string `json:"product"`
	PromotionCode string `json:"promotionCode"`
	Publisher     string `json:"publisher"`
}

// ContainerService complies with the ARM model of
// resource definition in a JSON template.
type ContainerService struct {
	ID       string                `json:"id"`
	Location string                `json:"location"`
	Name     string                `json:"name"`
	Plan     *ResourcePurchasePlan `json:"plan,omitempty"`
	Tags     map[string]string     `json:"tags"`
	Type     string                `json:"type"`

	Properties *Properties `json:"properties,omitempty"`
}

// AgentPoolResource complies with the ARM model of
// agentpool resource definition in a JSON template.
type AgentPoolResource struct {
	ID       string                `json:"id"`
	Location string                `json:"location"`
	Name     string                `json:"name"`
	Plan     *ResourcePurchasePlan `json:"plan,omitempty"`
	Tags     map[string]string     `json:"tags"`
	Type     string                `json:"type"`

	Properties *AgentPoolProfile `json:"properties,omitempty"`
}

// Properties represents the AKS cluster definition
type Properties struct {
	ClusterID               string
	ProvisioningState       ProvisioningState        `json:"provisioningState,omitempty"`
	OrchestratorProfile     *OrchestratorProfile     `json:"orchestratorProfile,omitempty"`
	MasterProfile           *MasterProfile           `json:"masterProfile,omitempty"`
	AgentPoolProfiles       []*AgentPoolProfile      `json:"agentPoolProfiles,omitempty"`
	LinuxProfile            *LinuxProfile            `json:"linuxProfile,omitempty"`
	WindowsProfile          *WindowsProfile          `json:"windowsProfile,omitempty"`
	ExtensionProfiles       []*ExtensionProfile      `json:"extensionProfiles"`
	DiagnosticsProfile      *DiagnosticsProfile      `json:"diagnosticsProfile,omitempty"`
	JumpboxProfile          *JumpboxProfile          `json:"jumpboxProfile,omitempty"`
	ServicePrincipalProfile *ServicePrincipalProfile `json:"servicePrincipalProfile,omitempty"`
	CertificateProfile      *CertificateProfile      `json:"certificateProfile,omitempty"`
	AADProfile              *AADProfile              `json:"aadProfile,omitempty"`
	CustomProfile           *CustomProfile           `json:"customProfile,omitempty"`
	HostedMasterProfile     *HostedMasterProfile     `json:"hostedMasterProfile,omitempty"`
	AddonProfiles           map[string]AddonProfile  `json:"addonProfiles,omitempty"`
	FeatureFlags            *FeatureFlags            `json:"featureFlags,omitempty"`
	CustomCloudProfile      *CustomCloudProfile      `json:"customCloudProfile,omitempty"`
}

// ClusterMetadata represents the metadata of the AKS cluster.
type ClusterMetadata struct {
	SubnetName                 string `json:"subnetName,omitempty"`
	VNetResourceGroupName      string `json:"vnetResourceGroupName,omitempty"`
	VirtualNetworkName         string `json:"virtualNetworkName,omitempty"`
	SecurityGroupName          string `json:"securityGroupName,omitempty"`
	RouteTableName             string `json:"routeTableName,omitempty"`
	PrimaryAvailabilitySetName string `json:"primaryAvailabilitySetName,omitempty"`
	PrimaryScaleSetName        string `json:"primaryScaleSetName,omitempty"`
	ResourcePrefix             string `json:"resourcePrefix,omitempty"`
}

// AddonProfile represents an addon for managed cluster
type AddonProfile struct {
	Enabled bool              `json:"enabled"`
	Config  map[string]string `json:"config"`
}

// FeatureFlags defines feature-flag restricted functionality
type FeatureFlags struct {
	EnableCSERunInBackground bool `json:"enableCSERunInBackground,omitempty"`
	BlockOutboundInternet    bool `json:"blockOutboundInternet,omitempty"`
	EnableIPv6DualStack      bool `json:"enableIPv6DualStack,omitempty"`
}

// ServicePrincipalProfile contains the client and secret used by the cluster for Azure Resource CRUD
type ServicePrincipalProfile struct {
	ClientID          string             `json:"clientId"`
	Secret            string             `json:"secret,omitempty" conform:"redact"`
	ObjectID          string             `json:"objectId,omitempty"`
	KeyvaultSecretRef *KeyvaultSecretRef `json:"keyvaultSecretRef,omitempty"`
}

// KeyvaultSecretRef specifies path to the Azure keyvault along with secret name and (optionaly) version
// for Service Principal's secret
type KeyvaultSecretRef struct {
	VaultID       string `json:"vaultID"`
	SecretName    string `json:"secretName"`
	SecretVersion string `json:"version,omitempty"`
}

// CertificateProfile represents the definition of the master cluster
type CertificateProfile struct {
	// CaCertificate is the certificate authority certificate.
	CaCertificate string `json:"caCertificate,omitempty" conform:"redact"`
	// CaPrivateKey is the certificate authority key.
	CaPrivateKey string `json:"caPrivateKey,omitempty" conform:"redact"`
	// ApiServerCertificate is the rest api server certificate, and signed by the CA
	APIServerCertificate string `json:"apiServerCertificate,omitempty" conform:"redact"`
	// ApiServerPrivateKey is the rest api server private key, and signed by the CA
	APIServerPrivateKey string `json:"apiServerPrivateKey,omitempty" conform:"redact"`
	// ClientCertificate is the certificate used by the client kubelet services and signed by the CA
	ClientCertificate string `json:"clientCertificate,omitempty" conform:"redact"`
	// ClientPrivateKey is the private key used by the client kubelet services and signed by the CA
	ClientPrivateKey string `json:"clientPrivateKey,omitempty" conform:"redact"`
	// KubeConfigCertificate is the client certificate used for kubectl cli and signed by the CA
	KubeConfigCertificate string `json:"kubeConfigCertificate,omitempty" conform:"redact"`
	// KubeConfigPrivateKey is the client private key used for kubectl cli and signed by the CA
	KubeConfigPrivateKey string `json:"kubeConfigPrivateKey,omitempty" conform:"redact"`
	// EtcdServerCertificate is the server certificate for etcd, and signed by the CA
	EtcdServerCertificate string `json:"etcdServerCertificate,omitempty" conform:"redact"`
	// EtcdServerPrivateKey is the server private key for etcd, and signed by the CA
	EtcdServerPrivateKey string `json:"etcdServerPrivateKey,omitempty" conform:"redact"`
	// EtcdClientCertificate is etcd client certificate, and signed by the CA
	EtcdClientCertificate string `json:"etcdClientCertificate,omitempty" conform:"redact"`
	// EtcdClientPrivateKey is the etcd client private key, and signed by the CA
	EtcdClientPrivateKey string `json:"etcdClientPrivateKey,omitempty" conform:"redact"`
	// EtcdPeerCertificates is list of etcd peer certificates, and signed by the CA
	EtcdPeerCertificates []string `json:"etcdPeerCertificates,omitempty" conform:"redact"`
	// EtcdPeerPrivateKeys is list of etcd peer private keys, and signed by the CA
	EtcdPeerPrivateKeys []string `json:"etcdPeerPrivateKeys,omitempty" conform:"redact"`
}

// LinuxProfile represents the linux parameters passed to the cluster
type LinuxProfile struct {
	AdminUsername string `json:"adminUsername"`
	SSH           struct {
		PublicKeys []PublicKey `json:"publicKeys"`
	} `json:"ssh"`
	Secrets               []KeyVaultSecrets   `json:"secrets,omitempty"`
	Distro                Distro              `json:"distro,omitempty"`
	ScriptRootURL         string              `json:"scriptroot,omitempty"`
	CustomSearchDomain    *CustomSearchDomain `json:"customSearchDomain,omitempty"`
	CustomNodesDNS        *CustomNodesDNS     `json:"CustomNodesDNS,omitempty"`
	IsSSHKeyAutoGenerated *bool               `json:"isSSHKeyAutoGenerated,omitempty"`
}

// PublicKey represents an SSH key for LinuxProfile
type PublicKey struct {
	KeyData string `json:"keyData"`
}

// CustomSearchDomain represents the Search Domain when the custom vnet has a windows server DNS as a nameserver.
type CustomSearchDomain struct {
	Name          string `json:"name,omitempty"`
	RealmUser     string `json:"realmUser,omitempty"`
	RealmPassword string `json:"realmPassword,omitempty"`
}

// CustomNodesDNS represents the Search Domain when the custom vnet for a custom DNS as a nameserver.
type CustomNodesDNS struct {
	DNSServer string `json:"dnsServer,omitempty"`
}

// WindowsProfile represents the windows parameters passed to the cluster
type WindowsProfile struct {
	AdminUsername          string            `json:"adminUsername"`
	AdminPassword          string            `json:"adminPassword" conform:"redact"`
	ImageVersion           string            `json:"imageVersion"`
	WindowsImageSourceURL  string            `json:"windowsImageSourceURL"`
	WindowsPublisher       string            `json:"windowsPublisher"`
	WindowsOffer           string            `json:"windowsOffer"`
	WindowsSku             string            `json:"windowsSku"`
	WindowsDockerVersion   string            `json:"windowsDockerVersion"`
	Secrets                []KeyVaultSecrets `json:"secrets,omitempty"`
	SSHEnabled             bool              `json:"sshEnabled,omitempty"`
	EnableAutomaticUpdates *bool             `json:"enableAutomaticUpdates,omitempty"`
}

// ProvisioningState represents the current state of container service resource.
type ProvisioningState string

const (
	// Creating means ContainerService resource is being created.
	Creating ProvisioningState = "Creating"
	// Updating means an existing ContainerService resource is being updated
	Updating ProvisioningState = "Updating"
	// Scaling means an existing ContainerService resource is being scaled only
	Scaling ProvisioningState = "Scaling"
	// Failed means resource is in failed state
	Failed ProvisioningState = "Failed"
	// Succeeded means resource created succeeded during last create/update
	Succeeded ProvisioningState = "Succeeded"
	// Deleting means resource is in the process of being deleted
	Deleting ProvisioningState = "Deleting"
	// Migrating means resource is being migrated from one subscription or
	// resource group to another
	Migrating ProvisioningState = "Migrating"
	// Upgrading means an existing ContainerService resource is being upgraded
	Upgrading ProvisioningState = "Upgrading"
)

// OrchestratorProfile contains Orchestrator properties
type OrchestratorProfile struct {
	OrchestratorType    string            `json:"orchestratorType"`
	OrchestratorVersion string            `json:"orchestratorVersion"`
	KubernetesConfig    *KubernetesConfig `json:"kubernetesConfig,omitempty"`
	DcosConfig          *DcosConfig       `json:"dcosConfig,omitempty"`
}

// OrchestratorVersionProfile contains information of a supported orchestrator version:
type OrchestratorVersionProfile struct {
	// Orchestrator type and version
	OrchestratorProfile
	// Whether this orchestrator version is deployed by default if orchestrator release is not specified
	Default bool `json:"default,omitempty"`
	// List of available upgrades for this orchestrator version
	Upgrades []*OrchestratorProfile `json:"upgrades,omitempty"`
}

// KubernetesContainerSpec defines configuration for a container spec
type KubernetesContainerSpec struct {
	Name           string `json:"name,omitempty"`
	Image          string `json:"image,omitempty"`
	CPURequests    string `json:"cpuRequests,omitempty"`
	MemoryRequests string `json:"memoryRequests,omitempty"`
	CPULimits      string `json:"cpuLimits,omitempty"`
	MemoryLimits   string `json:"memoryLimits,omitempty"`
}

// KubernetesAddon defines a list of addons w/ configuration to include with the cluster deployment
type KubernetesAddon struct {
	Name       string                    `json:"name,omitempty"`
	Enabled    *bool                     `json:"enabled,omitempty"`
	Containers []KubernetesContainerSpec `json:"containers,omitempty"`
	Config     map[string]string         `json:"config,omitempty"`
	Data       string                    `json:"data,omitempty"`
}

// IsEnabled returns true if the addon is enabled
func (a *KubernetesAddon) IsEnabled() bool {
	if a.Enabled == nil {
		return false
	}
	return *a.Enabled
}

// GetAddonContainersIndexByName returns the KubernetesAddon containers index with the name `containerName`
func (a KubernetesAddon) GetAddonContainersIndexByName(containerName string) int {
	for i := range a.Containers {
		if a.Containers[i].Name == containerName {
			return i
		}
	}
	return -1
}

// PrivateCluster defines the configuration for a private cluster
type PrivateCluster struct {
	Enabled        *bool                  `json:"enabled,omitempty"`
	JumpboxProfile *PrivateJumpboxProfile `json:"jumpboxProfile,omitempty"`
}

// PrivateJumpboxProfile represents a jumpbox definition
type PrivateJumpboxProfile struct {
	Name           string `json:"name" validate:"required"`
	VMSize         string `json:"vmSize" validate:"required"`
	OSDiskSizeGB   int    `json:"osDiskSizeGB,omitempty" validate:"min=0,max=1023"`
	Username       string `json:"username,omitempty"`
	PublicKey      string `json:"publicKey" validate:"required"`
	StorageProfile string `json:"storageProfile,omitempty"`
}

// CloudProviderConfig contains the KubernetesConfig properties specific to the Cloud Provider
type CloudProviderConfig struct {
	CloudProviderBackoff         *bool  `json:"cloudProviderBackoff,omitempty"`
	CloudProviderBackoffRetries  int    `json:"cloudProviderBackoffRetries,omitempty"`
	CloudProviderBackoffJitter   string `json:"cloudProviderBackoffJitter,omitempty"`
	CloudProviderBackoffDuration int    `json:"cloudProviderBackoffDuration,omitempty"`
	CloudProviderBackoffExponent string `json:"cloudProviderBackoffExponent,omitempty"`
	CloudProviderRateLimit       *bool  `json:"cloudProviderRateLimit,omitempty"`
	CloudProviderRateLimitQPS    string `json:"cloudProviderRateLimitQPS,omitempty"`
	CloudProviderRateLimitBucket int    `json:"cloudProviderRateLimitBucket,omitempty"`
}

// KubernetesConfigDeprecated are properties that are no longer operable and will be ignored
// TODO use this when strict JSON checking accommodates struct embedding
type KubernetesConfigDeprecated struct {
	NonMasqueradeCidr                string `json:"nonMasqueradeCidr,omitempty"`
	NodeStatusUpdateFrequency        string `json:"nodeStatusUpdateFrequency,omitempty"`
	HardEvictionThreshold            string `json:"hardEvictionThreshold,omitempty"`
	CtrlMgrNodeMonitorGracePeriod    string `json:"ctrlMgrNodeMonitorGracePeriod,omitempty"`
	CtrlMgrPodEvictionTimeout        string `json:"ctrlMgrPodEvictionTimeout,omitempty"`
	CtrlMgrRouteReconciliationPeriod string `json:"ctrlMgrRouteReconciliationPeriod,omitempty"`
}

// KubeProxyMode is for iptables and ipvs (and future others)
type KubeProxyMode string

// We currently support ipvs and iptables
const (
	// KubeProxyModeIPTables is used to set the kube-proxy to iptables mode
	KubeProxyModeIPTables KubeProxyMode = "iptables"
	// KubeProxyModeIPVS is used to set the kube-proxy to ipvs mode
	KubeProxyModeIPVS KubeProxyMode = "ipvs"
)

// KubernetesConfig contains the Kubernetes config structure, containing
// Kubernetes specific configuration
type KubernetesConfig struct {
	KubernetesImageBase              string            `json:"kubernetesImageBase,omitempty"`
	ClusterSubnet                    string            `json:"clusterSubnet,omitempty"`
	NetworkPolicy                    string            `json:"networkPolicy,omitempty"`
	NetworkPlugin                    string            `json:"networkPlugin,omitempty"`
	ContainerRuntime                 string            `json:"containerRuntime,omitempty"`
	MaxPods                          int               `json:"maxPods,omitempty"`
	DockerBridgeSubnet               string            `json:"dockerBridgeSubnet,omitempty"`
	DNSServiceIP                     string            `json:"dnsServiceIP,omitempty"`
	ServiceCIDR                      string            `json:"serviceCidr,omitempty"`
	UseManagedIdentity               bool              `json:"useManagedIdentity,omitempty"`
	UserAssignedID                   string            `json:"userAssignedID,omitempty"`
	UserAssignedClientID             string            `json:"userAssignedClientID,omitempty"` //Note: cannot be provided in config. Used *only* for transferring this to azure.json.
	CustomHyperkubeImage             string            `json:"customHyperkubeImage,omitempty"`
	DockerEngineVersion              string            `json:"dockerEngineVersion,omitempty"` // Deprecated
	MobyVersion                      string            `json:"mobyVersion,omitempty"`
	ContainerdVersion                string            `json:"containerdVersion,omitempty"`
	CustomCcmImage                   string            `json:"customCcmImage,omitempty"` // Image for cloud-controller-manager
	UseCloudControllerManager        *bool             `json:"useCloudControllerManager,omitempty"`
	CustomWindowsPackageURL          string            `json:"customWindowsPackageURL,omitempty"`
	WindowsNodeBinariesURL           string            `json:"windowsNodeBinariesURL,omitempty"`
	UseInstanceMetadata              *bool             `json:"useInstanceMetadata,omitempty"`
	EnableRbac                       *bool             `json:"enableRbac,omitempty"`
	EnableSecureKubelet              *bool             `json:"enableSecureKubelet,omitempty"`
	EnableAggregatedAPIs             bool              `json:"enableAggregatedAPIs,omitempty"`
	PrivateCluster                   *PrivateCluster   `json:"privateCluster,omitempty"`
	GCHighThreshold                  int               `json:"gchighthreshold,omitempty"`
	GCLowThreshold                   int               `json:"gclowthreshold,omitempty"`
	EtcdVersion                      string            `json:"etcdVersion,omitempty"`
	EtcdDiskSizeGB                   string            `json:"etcdDiskSizeGB,omitempty"`
	EtcdEncryptionKey                string            `json:"etcdEncryptionKey,omitempty"`
	EnableDataEncryptionAtRest       *bool             `json:"enableDataEncryptionAtRest,omitempty"`
	EnableEncryptionWithExternalKms  *bool             `json:"enableEncryptionWithExternalKms,omitempty"`
	EnablePodSecurityPolicy          *bool             `json:"enablePodSecurityPolicy,omitempty"`
	Addons                           []KubernetesAddon `json:"addons,omitempty"`
	KubeletConfig                    map[string]string `json:"kubeletConfig,omitempty"`
	ControllerManagerConfig          map[string]string `json:"controllerManagerConfig,omitempty"`
	CloudControllerManagerConfig     map[string]string `json:"cloudControllerManagerConfig,omitempty"`
	APIServerConfig                  map[string]string `json:"apiServerConfig,omitempty"`
	SchedulerConfig                  map[string]string `json:"schedulerConfig,omitempty"`
	PodSecurityPolicyConfig          map[string]string `json:"podSecurityPolicyConfig,omitempty"` // Deprecated
	CloudProviderBackoff             *bool             `json:"cloudProviderBackoff,omitempty"`
	CloudProviderBackoffRetries      int               `json:"cloudProviderBackoffRetries,omitempty"`
	CloudProviderBackoffJitter       float64           `json:"cloudProviderBackoffJitter,omitempty"`
	CloudProviderBackoffDuration     int               `json:"cloudProviderBackoffDuration,omitempty"`
	CloudProviderBackoffExponent     float64           `json:"cloudProviderBackoffExponent,omitempty"`
	CloudProviderRateLimit           *bool             `json:"cloudProviderRateLimit,omitempty"`
	CloudProviderRateLimitQPS        float64           `json:"cloudProviderRateLimitQPS,omitempty"`
	CloudProviderRateLimitBucket     int               `json:"cloudProviderRateLimitBucket,omitempty"`
	NonMasqueradeCidr                string            `json:"nonMasqueradeCidr,omitempty"`
	NodeStatusUpdateFrequency        string            `json:"nodeStatusUpdateFrequency,omitempty"`
	HardEvictionThreshold            string            `json:"hardEvictionThreshold,omitempty"`
	CtrlMgrNodeMonitorGracePeriod    string            `json:"ctrlMgrNodeMonitorGracePeriod,omitempty"`
	CtrlMgrPodEvictionTimeout        string            `json:"ctrlMgrPodEvictionTimeout,omitempty"`
	CtrlMgrRouteReconciliationPeriod string            `json:"ctrlMgrRouteReconciliationPeriod,omitempty"`
	LoadBalancerSku                  string            `json:"loadBalancerSku,omitempty"`
	ExcludeMasterFromStandardLB      *bool             `json:"excludeMasterFromStandardLB,omitempty"`
	AzureCNIVersion                  string            `json:"azureCNIVersion,omitempty"`
	AzureCNIURLLinux                 string            `json:"azureCNIURLLinux,omitempty"`
	AzureCNIURLWindows               string            `json:"azureCNIURLWindows,omitempty"`
	KeyVaultSku                      string            `json:"keyVaultSku,omitempty"`
	MaximumLoadBalancerRuleCount     int               `json:"maximumLoadBalancerRuleCount,omitempty"`
	ProxyMode                        KubeProxyMode     `json:"kubeProxyMode,omitempty"`
	PrivateAzureRegistryServer       string            `json:"privateAzureRegistryServer,omitempty"`
	OutboundRuleIdleTimeoutInMinutes int32             `json:"outboundRuleIdleTimeoutInMinutes,omitempty"`
}

// CustomFile has source as the full absolute source path to a file and dest
// is the full absolute desired destination path to put the file on a master node
type CustomFile struct {
	Source string `json:"source,omitempty"`
	Dest   string `json:"dest,omitempty"`
}

// BootstrapProfile represents the definition of the DCOS bootstrap node used to deploy the cluster
type BootstrapProfile struct {
	VMSize       string `json:"vmSize,omitempty"`
	OSDiskSizeGB int    `json:"osDiskSizeGB,omitempty"`
	OAuthEnabled bool   `json:"oauthEnabled,omitempty"`
	StaticIP     string `json:"staticIP,omitempty"`
	Subnet       string `json:"subnet,omitempty"`
}

// DcosConfig Configuration for DC/OS
type DcosConfig struct {
	DcosBootstrapURL         string            `json:"dcosBootstrapURL,omitempty"`
	DcosWindowsBootstrapURL  string            `json:"dcosWindowsBootstrapURL,omitempty"`
	Registry                 string            `json:"registry,omitempty"`
	RegistryUser             string            `json:"registryUser,omitempty"`
	RegistryPass             string            `json:"registryPassword,omitempty"`
	DcosRepositoryURL        string            `json:"dcosRepositoryURL,omitempty"`        // For CI use, you need to specify
	DcosClusterPackageListID string            `json:"dcosClusterPackageListID,omitempty"` // all three of these items
	DcosProviderPackageID    string            `json:"dcosProviderPackageID,omitempty"`    // repo url is the location of the build,
	BootstrapProfile         *BootstrapProfile `json:"bootstrapProfile,omitempty"`
}

// HasPrivateRegistry returns if a private registry is specified
func (d *DcosConfig) HasPrivateRegistry() bool {
	return len(d.Registry) > 0
}

// HasBootstrap returns if a bootstrap profile is specified
func (d *DcosConfig) HasBootstrap() bool {
	return d.BootstrapProfile != nil
}

// MasterProfile represents the definition of the master cluster
type MasterProfile struct {
	Count                    int               `json:"count"`
	DNSPrefix                string            `json:"dnsPrefix"`
	SubjectAltNames          []string          `json:"subjectAltNames"`
	VMSize                   string            `json:"vmSize"`
	OSDiskSizeGB             int               `json:"osDiskSizeGB,omitempty"`
	VnetSubnetID             string            `json:"vnetSubnetID,omitempty"`
	VnetCidr                 string            `json:"vnetCidr,omitempty"`
	AgentVnetSubnetID        string            `json:"agentVnetSubnetID,omitempty"`
	FirstConsecutiveStaticIP string            `json:"firstConsecutiveStaticIP,omitempty"`
	Subnet                   string            `json:"subnet"`
	SubnetIPv6               string            `json:"subnetIPv6"`
	IPAddressCount           int               `json:"ipAddressCount,omitempty"`
	StorageProfile           string            `json:"storageProfile,omitempty"`
	HTTPSourceAddressPrefix  string            `json:"HTTPSourceAddressPrefix,omitempty"`
	OAuthEnabled             bool              `json:"oauthEnabled"`
	PreprovisionExtension    *Extension        `json:"preProvisionExtension"`
	Extensions               []Extension       `json:"extensions"`
	Distro                   Distro            `json:"distro,omitempty"`
	KubernetesConfig         *KubernetesConfig `json:"kubernetesConfig,omitempty"`
	ImageRef                 *ImageReference   `json:"imageReference,omitempty"`
	CustomFiles              *[]CustomFile     `json:"customFiles,omitempty"`
	AvailabilityProfile      string            `json:"availabilityProfile"`
	PlatformFaultDomainCount *int              `json:"platformFaultDomainCount"`
	AgentSubnet              string            `json:"agentSubnet,omitempty"`
	AvailabilityZones        []string          `json:"availabilityZones,omitempty"`
	SinglePlacementGroup     *bool             `json:"singlePlacementGroup,omitempty"`
	AuditDEnabled            *bool             `json:"auditDEnabled,omitempty"`
	CustomVMTags             map[string]string `json:"customVMTags,omitempty"`
	// Master LB public endpoint/FQDN with port
	// The format will be FQDN:2376
	// Not used during PUT, returned as part of GET
	FQDN string `json:"fqdn,omitempty"`
	// True: uses cosmos etcd endpoint instead of installing etcd on masters
	CosmosEtcd *bool `json:"cosmosEtcd,omitempty"`
}

// ImageReference represents a reference to an Image resource in Azure.
type ImageReference struct {
	Name           string `json:"name,omitempty"`
	ResourceGroup  string `json:"resourceGroup,omitempty"`
	SubscriptionID string `json:"subscriptionId,omitempty"`
	Gallery        string `json:"gallery,omitempty"`
	Version        string `json:"version,omitempty"`
}

// ExtensionProfile represents an extension definition
type ExtensionProfile struct {
	Name                           string             `json:"name"`
	Version                        string             `json:"version"`
	ExtensionParameters            string             `json:"extensionParameters,omitempty"`
	ExtensionParametersKeyVaultRef *KeyvaultSecretRef `json:"parametersKeyvaultSecretRef,omitempty"`
	RootURL                        string             `json:"rootURL,omitempty"`
	// This is only needed for preprovision extensions and it needs to be a bash script
	Script   string `json:"script,omitempty"`
	URLQuery string `json:"urlQuery,omitempty"`
}

// Extension represents an extension definition in the master or agentPoolProfile
type Extension struct {
	Name        string `json:"name"`
	SingleOrAll string `json:"singleOrAll"`
	Template    string `json:"template"`
}

// AgentPoolProfile represents an agent pool definition
type AgentPoolProfile struct {
	Name                                string               `json:"name"`
	Count                               int                  `json:"count"`
	VMSize                              string               `json:"vmSize"`
	OSDiskSizeGB                        int                  `json:"osDiskSizeGB,omitempty"`
	DNSPrefix                           string               `json:"dnsPrefix,omitempty"`
	OSType                              OSType               `json:"osType,omitempty"`
	Ports                               []int                `json:"ports,omitempty"`
	ProvisioningState                   ProvisioningState    `json:"provisioningState,omitempty"`
	AvailabilityProfile                 string               `json:"availabilityProfile"`
	PlatformFaultDomainCount            *int                 `json:"platformFaultDomainCount"`
	ScaleSetPriority                    string               `json:"scaleSetPriority,omitempty"`
	ScaleSetEvictionPolicy              string               `json:"scaleSetEvictionPolicy,omitempty"`
	StorageProfile                      string               `json:"storageProfile,omitempty"`
	DiskSizesGB                         []int                `json:"diskSizesGB,omitempty"`
	VnetSubnetID                        string               `json:"vnetSubnetID,omitempty"`
	Subnet                              string               `json:"subnet"`
	IPAddressCount                      int                  `json:"ipAddressCount,omitempty"`
	Distro                              Distro               `json:"distro,omitempty"`
	Role                                AgentPoolProfileRole `json:"role,omitempty"`
	AcceleratedNetworkingEnabled        *bool                `json:"acceleratedNetworkingEnabled,omitempty"`
	AcceleratedNetworkingEnabledWindows *bool                `json:"acceleratedNetworkingEnabledWindows,omitempty"`
	VMSSOverProvisioningEnabled         *bool                `json:"vmssOverProvisioningEnabled,omitempty"`
	FQDN                                string               `json:"fqdn,omitempty"`
	CustomNodeLabels                    map[string]string    `json:"customNodeLabels,omitempty"`
	PreprovisionExtension               *Extension           `json:"preProvisionExtension"`
	Extensions                          []Extension          `json:"extensions"`
	KubernetesConfig                    *KubernetesConfig    `json:"kubernetesConfig,omitempty"`
	OrchestratorVersion                 string               `json:"orchestratorVersion"`
	ImageRef                            *ImageReference      `json:"imageReference,omitempty"`
	MaxCount                            *int                 `json:"maxCount,omitempty"`
	MinCount                            *int                 `json:"minCount,omitempty"`
	EnableAutoScaling                   *bool                `json:"enableAutoScaling,omitempty"`
	AvailabilityZones                   []string             `json:"availabilityZones,omitempty"`
	SinglePlacementGroup                *bool                `json:"singlePlacementGroup,omitempty"`
	VnetCidrs                           []string             `json:"vnetCidrs,omitempty"`
	PreserveNodesProperties             *bool                `json:"preserveNodesProperties,omitempty"`
	WindowsNameVersion                  string               `json:"windowsNameVersion,omitempty"`
	EnableVMSSNodePublicIP              *bool                `json:"enableVMSSNodePublicIP,omitempty"`
	LoadBalancerBackendAddressPoolIDs   []string             `json:"loadBalancerBackendAddressPoolIDs,omitempty"`
	AuditDEnabled                       *bool                `json:"auditDEnabled,omitempty"`
	CustomVMTags                        map[string]string    `json:"customVMTags,omitempty"`
}

// AgentPoolProfileRole represents an agent role
type AgentPoolProfileRole string

// DiagnosticsProfile setting to enable/disable capturing
// diagnostics for VMs hosting container cluster.
type DiagnosticsProfile struct {
	VMDiagnostics *VMDiagnostics `json:"vmDiagnostics"`
}

// VMDiagnostics contains settings to on/off boot diagnostics collection
// in RD Host
type VMDiagnostics struct {
	Enabled bool `json:"enabled"`

	// Specifies storage account Uri where Boot Diagnostics (CRP &
	// VMSS BootDiagostics) and VM Diagnostics logs (using Linux
	// Diagnostics Extension) will be stored. Uri will be of standard
	// blob domain. i.e. https://storageaccount.blob.core.windows.net/
	// This field is readonly as ACS RP will create a storage account
	// for the customer.
	StorageURL *neturl.URL `json:"storageUrl"`
}

// JumpboxProfile describes properties of the jumpbox setup
// in the AKS container cluster.
type JumpboxProfile struct {
	OSType    OSType `json:"osType"`
	DNSPrefix string `json:"dnsPrefix"`

	// Jumpbox public endpoint/FQDN with port
	// The format will be FQDN:2376
	// Not used during PUT, returned as part of GET
	FQDN string `json:"fqdn,omitempty"`
}

// KeyVaultSecrets specifies certificates to install on the pool
// of machines from a given key vault
// the key vault specified must have been granted read permissions to CRP
type KeyVaultSecrets struct {
	SourceVault       *KeyVaultID           `json:"sourceVault,omitempty"`
	VaultCertificates []KeyVaultCertificate `json:"vaultCertificates,omitempty"`
}

// KeyVaultID specifies a key vault
type KeyVaultID struct {
	ID string `json:"id,omitempty"`
}

// KeyVaultCertificate specifies a certificate to install
// On Linux, the certificate file is placed under the /var/lib/waagent directory
// with the file name <UppercaseThumbprint>.crt for the X509 certificate file
// and <UppercaseThumbprint>.prv for the private key. Both of these files are .pem formatted.
// On windows the certificate will be saved in the specified store.
type KeyVaultCertificate struct {
	CertificateURL   string `json:"certificateUrl,omitempty"`
	CertificateStore string `json:"certificateStore,omitempty"`
}

// OSType represents OS types of agents
type OSType string

// Distro represents Linux distro to use for Linux VMs
type Distro string

// HostedMasterProfile defines properties for a hosted master
type HostedMasterProfile struct {
	// Master public endpoint/FQDN with port
	// The format will be FQDN:2376
	// Not used during PUT, returned as part of GETFQDN
	FQDN      string `json:"fqdn,omitempty"`
	DNSPrefix string `json:"dnsPrefix"`
	// Subnet holds the CIDR which defines the Azure Subnet in which
	// Agents will be provisioned. This is stored on the HostedMasterProfile
	// and will become `masterSubnet` in the compiled template.
	Subnet string `json:"subnet"`
	// ApiServerWhiteListRange is a comma delimited CIDR which is whitelisted to AKS
	APIServerWhiteListRange *string `json:"apiServerWhiteListRange"`
	IPMasqAgent             bool    `json:"ipMasqAgent"`
}

// AuthenticatorType represents the authenticator type the cluster was
// set up with.
type AuthenticatorType string

const (
	// OIDC represent cluster setup in OIDC auth mode
	OIDC AuthenticatorType = "oidc"
	// Webhook represent cluster setup in wehhook auth mode
	Webhook AuthenticatorType = "webhook"
)

// AADProfile specifies attributes for AAD integration
type AADProfile struct {
	// The client AAD application ID.
	ClientAppID string `json:"clientAppID,omitempty"`
	// The server AAD application ID.
	ServerAppID string `json:"serverAppID,omitempty"`
	// The server AAD application secret
	ServerAppSecret string `json:"serverAppSecret,omitempty" conform:"redact"`
	// The AAD tenant ID to use for authentication.
	// If not specified, will use the tenant of the deployment subscription.
	// Optional
	TenantID string `json:"tenantID,omitempty"`
	// The Azure Active Directory Group Object ID that will be assigned the
	// cluster-admin RBAC role.
	// Optional
	AdminGroupID string `json:"adminGroupID,omitempty"`
	// The authenticator to use, either "oidc" or "webhook".
	Authenticator AuthenticatorType `json:"authenticator"`
}

// CustomProfile specifies custom properties that are used for
// cluster instantiation.  Should not be used by most users.
type CustomProfile struct {
	Orchestrator string `json:"orchestrator,omitempty"`
}

// VlabsARMContainerService is the type we read and write from file
// needed because the json that is sent to ARM and aks-engine
// is different from the json that the ACS RP Api gets from ARM
type VlabsARMContainerService struct {
	TypeMeta
	*vlabs.ContainerService
}

// V20160330ARMContainerService is the type we read and write from file
// needed because the json that is sent to ARM and aks-engine
// is different from the json that the ACS RP Api gets from ARM
type V20160330ARMContainerService struct {
	TypeMeta
	*v20160330.ContainerService
}

// V20160930ARMContainerService is the type we read and write from file
// needed because the json that is sent to ARM and aks-engine
// is different from the json that the ACS RP Api gets from ARM
type V20160930ARMContainerService struct {
	TypeMeta
	*v20160930.ContainerService
}

// V20170131ARMContainerService is the type we read and write from file
// needed because the json that is sent to ARM and aks-engine
// is different from the json that the ACS RP Api gets from ARM
type V20170131ARMContainerService struct {
	TypeMeta
	*v20170131.ContainerService
}

// V20170701ARMContainerService is the type we read and write from file
// needed because the json that is sent to ARM and aks-engine
// is different from the json that the ACS RP Api gets from ARM
type V20170701ARMContainerService struct {
	TypeMeta
	*v20170701.ContainerService
}

// V20170831ARMManagedContainerService is the type we read and write from file
// needed because the json that is sent to ARM and aks-engine
// is different from the json that the ACS RP Api gets from ARM
type V20170831ARMManagedContainerService struct {
	TypeMeta
	*v20170831.ManagedCluster
}

// V20180331ARMManagedContainerService is the type we read and write from file
// needed because the json that is sent to ARM and aks-engine
// is different from the json that the ACS RP Api gets from ARM
type V20180331ARMManagedContainerService struct {
	TypeMeta
	*v20180331.ManagedCluster
}

// AzureStackMetadataEndpoints is the type for Azure Stack metadata endpoints
type AzureStackMetadataEndpoints struct {
	GalleryEndpoint string                            `json:"galleryEndpoint,omitempty"`
	GraphEndpoint   string                            `json:"graphEndpoint,omitempty"`
	PortalEndpoint  string                            `json:"portalEndpoint,omitempty"`
	Authentication  *AzureStackMetadataAuthentication `json:"authentication,omitempty"`
}

// AzureStackMetadataAuthentication is the type for Azure Stack metadata authentication endpoints
type AzureStackMetadataAuthentication struct {
	LoginEndpoint string   `json:"loginEndpoint,omitempty"`
	Audiences     []string `json:"audiences,omitempty"`
}

// DependenciesLocation represents location to retrieve the dependencies.
type DependenciesLocation string

// CustomCloudProfile represents the custom cloud profile
type CustomCloudProfile struct {
	Environment                *azure.Environment          `json:"environment,omitempty"`
	AzureEnvironmentSpecConfig *AzureEnvironmentSpecConfig `json:"azureEnvironmentSpecConfig,omitempty"`
	IdentitySystem             string                      `json:"identitySystem,omitempty"`
	AuthenticationMethod       string                      `json:"authenticationMethod,omitempty"`
	DependenciesLocation       DependenciesLocation        `json:"dependenciesLocation,omitempty"`
	PortalURL                  string                      `json:"portalURL,omitempty"`
}

// HasCoreOS returns true if the cluster contains coreos nodes
func (p *Properties) HasCoreOS() bool {
	for _, agentPoolProfile := range p.AgentPoolProfiles {
		if agentPoolProfile.Distro == CoreOS {
			return true
		}
	}
	return false
}

// HasWindows returns true if the cluster contains windows
func (p *Properties) HasWindows() bool {
	for _, agentPoolProfile := range p.AgentPoolProfiles {
		if agentPoolProfile.OSType == Windows {
			return true
		}
	}
	return false
}

// HasManagedDisks returns true if the cluster contains Managed Disks
func (p *Properties) HasManagedDisks() bool {
	if p.MasterProfile != nil && p.MasterProfile.StorageProfile == ManagedDisks {
		return true
	}
	for _, agentPoolProfile := range p.AgentPoolProfiles {
		if agentPoolProfile.StorageProfile == ManagedDisks {
			return true
		}
	}
	if p.OrchestratorProfile != nil && p.OrchestratorProfile.KubernetesConfig != nil && p.OrchestratorProfile.KubernetesConfig.PrivateJumpboxProvision() && p.OrchestratorProfile.KubernetesConfig.PrivateCluster.JumpboxProfile.StorageProfile == ManagedDisks {
		return true
	}
	return false
}

// HasStorageAccountDisks returns true if the cluster contains Storage Account Disks
func (p *Properties) HasStorageAccountDisks() bool {
	if p.MasterProfile != nil && p.MasterProfile.StorageProfile == StorageAccount {
		return true
	}
	for _, agentPoolProfile := range p.AgentPoolProfiles {
		if agentPoolProfile.StorageProfile == StorageAccount {
			return true
		}
	}
	if p.OrchestratorProfile != nil && p.OrchestratorProfile.KubernetesConfig != nil && p.OrchestratorProfile.KubernetesConfig.PrivateJumpboxProvision() && p.OrchestratorProfile.KubernetesConfig.PrivateCluster.JumpboxProfile.StorageProfile == StorageAccount {
		return true
	}
	return false
}

// HasStorageAccountDisks returns true if the cluster contains agent pools with Ephemeral Disks
func (p *Properties) HasEphemeralDisks() bool {
	for _, agentPoolProfile := range p.AgentPoolProfiles {
		if agentPoolProfile.StorageProfile == Ephemeral {
			return true
		}
	}
	return false
}

// TotalNodes returns the total number of nodes in the cluster configuration
func (p *Properties) TotalNodes() int {
	var totalNodes int
	if p.MasterProfile != nil {
		totalNodes = p.MasterProfile.Count
	}
	for _, pool := range p.AgentPoolProfiles {
		totalNodes += pool.Count
	}
	return totalNodes
}

// HasVMSSAgentPool returns true if the cluster contains Virtual Machine Scale Sets agent pools
func (p *Properties) HasVMSSAgentPool() bool {
	for _, agentPoolProfile := range p.AgentPoolProfiles {
		if agentPoolProfile.AvailabilityProfile == VirtualMachineScaleSets {
			return true
		}
	}
	return false
}

// K8sOrchestratorName returns the 3 character orchestrator code for kubernetes-based clusters.
func (p *Properties) K8sOrchestratorName() string {
	if p.OrchestratorProfile.IsKubernetes() {
		if p.HostedMasterProfile != nil {
			return DefaultHostedProfileMasterName
		}
		return DefaultOrchestratorName
	}
	return ""
}

// GetAgentPoolIndexByName returns the index of the provided agentpool.
func (p *Properties) GetAgentPoolIndexByName(name string) int {
	index := -1
	for i, profile := range p.AgentPoolProfiles {
		if profile.Name == name {
			index = i
			break
		}
	}
	return index
}

// GetAgentVMPrefix returns the VM prefix for an agentpool.
func (p *Properties) GetAgentVMPrefix(a *AgentPoolProfile, index int) string {
	nameSuffix := p.GetClusterID()
	vmPrefix := ""
	if index != -1 {
		if a.IsWindows() {
			if a.WindowsNameVersion == "v2" {
				vmPrefix = p.K8sOrchestratorName() + a.Name
			} else {
				vmPrefix = nameSuffix[:4] + p.K8sOrchestratorName() + fmt.Sprintf("%02d", index)
			}
		} else {
			vmPrefix = p.K8sOrchestratorName() + "-" + a.Name + "-" + nameSuffix + "-"
			if a.IsVirtualMachineScaleSets() {
				vmPrefix += "vmss"
			}
		}
	}
	return vmPrefix
}

// AnyAgentUsesVirtualMachineScaleSets checks whether any of the agents in the AgentPool use VMSS or not
func (p *Properties) AnyAgentUsesVirtualMachineScaleSets() bool {
	for _, agentProfile := range p.AgentPoolProfiles {
		if agentProfile.IsVirtualMachineScaleSets() {
			return true
		}
	}
	return false
}

// AnyAgentUsesAvailabilitySets checks whether any of the agents in the AgentPool use VMAS or not
func (p *Properties) AnyAgentUsesAvailabilitySets() bool {
	for _, agentProfile := range p.AgentPoolProfiles {
		if agentProfile.IsAvailabilitySets() {
			return true
		}
	}
	return false
}

// AnyAgentIsLinux checks whether any of the agents in the AgentPools are linux
func (p *Properties) AnyAgentIsLinux() bool {
	for _, agentProfile := range p.AgentPoolProfiles {
		if agentProfile.IsLinux() {
			return true
		}
	}
	return false
}

// GetMasterVMPrefix returns the prefix of master VMs
func (p *Properties) GetMasterVMPrefix() string {
	return p.K8sOrchestratorName() + "-master-" + p.GetClusterID() + "-"
}

// GetResourcePrefix returns the prefix to use for naming cluster resources
func (p *Properties) GetResourcePrefix() string {
	if p.IsHostedMasterProfile() {
		return p.K8sOrchestratorName() + "-agentpool-" + p.GetClusterID() + "-"
	}
	return p.K8sOrchestratorName() + "-master-" + p.GetClusterID() + "-"

}

// GetRouteTableName returns the route table name of the cluster.
func (p *Properties) GetRouteTableName() string {
	return p.GetResourcePrefix() + "routetable"
}

// GetNSGName returns the name of the network security group of the cluster.
func (p *Properties) GetNSGName() string {
	return p.GetResourcePrefix() + "nsg"
}

// GetPrimaryAvailabilitySetName returns the name of the primary availability set of the cluster
func (p *Properties) GetPrimaryAvailabilitySetName() string {
	if len(p.AgentPoolProfiles) > 0 {
		if p.AgentPoolProfiles[0].AvailabilityProfile == AvailabilitySet {
			return p.AgentPoolProfiles[0].Name + "-availabilitySet-" + p.GetClusterID()
		}
	}
	return ""
}

// GetPrimaryScaleSetName returns the name of the primary scale set node of the cluster
func (p *Properties) GetPrimaryScaleSetName() string {
	if len(p.AgentPoolProfiles) > 0 {
		if p.AgentPoolProfiles[0].AvailabilityProfile == VirtualMachineScaleSets {
			return p.GetAgentVMPrefix(p.AgentPoolProfiles[0], 0)
		}
	}
	return ""
}

// IsHostedMasterProfile returns true if the cluster has a hosted master
func (p *Properties) IsHostedMasterProfile() bool {
	return p.HostedMasterProfile != nil
}

// IsIPMasqAgentEnabled returns true if the cluster has a hosted master and IpMasqAgent is disabled
func (p *Properties) IsIPMasqAgentEnabled() bool {
	if p.HostedMasterProfile != nil {
		return p.HostedMasterProfile.IPMasqAgent
	}
	return p.OrchestratorProfile.KubernetesConfig.IsIPMasqAgentEnabled()
}

// GetVNetResourceGroupName returns the virtual network resource group name of the cluster
func (p *Properties) GetVNetResourceGroupName() string {
	var vnetResourceGroupName string
	if p.IsHostedMasterProfile() && p.AreAgentProfilesCustomVNET() {
		vnetResourceGroupName = strings.Split(p.AgentPoolProfiles[0].VnetSubnetID, "/")[DefaultVnetResourceGroupSegmentIndex]
	} else if !p.IsHostedMasterProfile() && p.MasterProfile.IsCustomVNET() {
		vnetResourceGroupName = strings.Split(p.MasterProfile.VnetSubnetID, "/")[DefaultVnetResourceGroupSegmentIndex]
	}
	return vnetResourceGroupName
}

// GetVirtualNetworkName returns the virtual network name of the cluster
func (p *Properties) GetVirtualNetworkName() string {
	var vnetName string
	if p.IsHostedMasterProfile() && p.AreAgentProfilesCustomVNET() {
		vnetName = strings.Split(p.AgentPoolProfiles[0].VnetSubnetID, "/")[DefaultVnetNameResourceSegmentIndex]
	} else if !p.IsHostedMasterProfile() && p.MasterProfile.IsCustomVNET() {
		vnetName = strings.Split(p.MasterProfile.VnetSubnetID, "/")[DefaultVnetNameResourceSegmentIndex]
	} else {
		vnetName = p.K8sOrchestratorName() + "-vnet-" + p.GetClusterID()
	}
	return vnetName
}

// GetSubnetName returns the subnet name of the cluster based on its current configuration.
func (p *Properties) GetSubnetName() string {
	var subnetName string
	if p.IsHostedMasterProfile() {
		if p.AreAgentProfilesCustomVNET() {
			subnetName = strings.Split(p.AgentPoolProfiles[0].VnetSubnetID, "/")[DefaultSubnetNameResourceSegmentIndex]
		} else {
			subnetName = p.K8sOrchestratorName() + "-subnet"
		}
	} else {
		if p.MasterProfile.IsCustomVNET() {
			subnetName = strings.Split(p.MasterProfile.VnetSubnetID, "/")[DefaultSubnetNameResourceSegmentIndex]
		} else {
			subnetName = p.K8sOrchestratorName() + "-subnet"
		}
	}
	return subnetName
}

// AreAgentProfilesCustomVNET returns true if all of the agent profiles in the clusters are configured with VNET.
func (p *Properties) AreAgentProfilesCustomVNET() bool {
	if p.AgentPoolProfiles != nil {
		for _, agentPoolProfile := range p.AgentPoolProfiles {
			if !agentPoolProfile.IsCustomVNET() {
				return false
			}
		}
		return true
	}
	return false
}

// GetClusterID creates a unique 8 string cluster ID.
func (p *Properties) GetClusterID() string {
	var mutex = &sync.Mutex{}
	if p.ClusterID == "" {
		uniqueNameSuffixSize := 8
		// the name suffix uniquely identifies the cluster and is generated off a hash
		// from the master dns name
		h := fnv.New64a()
		if p.MasterProfile != nil {
			h.Write([]byte(p.MasterProfile.DNSPrefix))
		} else if p.HostedMasterProfile != nil {
			h.Write([]byte(p.HostedMasterProfile.DNSPrefix))
		} else if len(p.AgentPoolProfiles) > 0 {
			h.Write([]byte(p.AgentPoolProfiles[0].Name))
		}
		r := rand.New(rand.NewSource(int64(h.Sum64())))
		mutex.Lock()
		p.ClusterID = fmt.Sprintf("%08d", r.Uint32())[:uniqueNameSuffixSize]
		mutex.Unlock()
	}
	return p.ClusterID
}

// GetClusterMetadata returns a instance of the struct type api.ClusterMetadata.
func (p *Properties) GetClusterMetadata() *ClusterMetadata {
	return &ClusterMetadata{
		SubnetName:                 p.GetSubnetName(),
		VNetResourceGroupName:      p.GetVNetResourceGroupName(),
		VirtualNetworkName:         p.GetVirtualNetworkName(),
		SecurityGroupName:          p.GetNSGName(),
		RouteTableName:             p.GetRouteTableName(),
		PrimaryAvailabilitySetName: p.GetPrimaryAvailabilitySetName(),
		PrimaryScaleSetName:        p.GetPrimaryScaleSetName(),
		ResourcePrefix:             p.GetResourcePrefix(),
	}
}

// HasZonesForAllAgentPools returns true if all of the agent pools have zones
func (p *Properties) HasZonesForAllAgentPools() bool {
	if len(p.AgentPoolProfiles) > 0 {
		for _, ap := range p.AgentPoolProfiles {
			if !ap.HasAvailabilityZones() {
				return false
			}
		}
		return true
	}
	return false
}

// IsVHDDistroForAllNodes returns true if all of the agent pools plus masters are running the VHD image
func (p *Properties) IsVHDDistroForAllNodes() bool {
	if len(p.AgentPoolProfiles) > 0 {
		for _, ap := range p.AgentPoolProfiles {
			if !ap.IsVHDDistro() {
				return false
			}
		}
	}
	if p.MasterProfile != nil {
		return p.MasterProfile.IsVHDDistro()
	}
	return true
}

// IsUbuntuDistroForAllNodes returns true if all of the agent pools plus masters are running the base Ubuntu image
func (p *Properties) IsUbuntuDistroForAllNodes() bool {
	if len(p.AgentPoolProfiles) > 0 {
		for _, ap := range p.AgentPoolProfiles {
			if ap.Distro != Ubuntu && ap.Distro != Ubuntu1804 {
				return false
			}
		}
	}
	if p.MasterProfile != nil {
		return p.MasterProfile.Distro == Ubuntu || p.MasterProfile.Distro == Ubuntu1804
	}
	return true
}

// HasUbuntuDistroNodes returns true if any of the agent pools or masters are running the base Ubuntu image
func (p *Properties) HasUbuntuDistroNodes() bool {
	if len(p.AgentPoolProfiles) > 0 {
		for _, ap := range p.AgentPoolProfiles {
			if ap.Distro == Ubuntu || ap.Distro == Ubuntu1804 {
				return true
			}
		}
	}
	if p.MasterProfile != nil {
		return p.MasterProfile.Distro == Ubuntu || p.MasterProfile.Distro == Ubuntu1804
	}
	return false
}

// HasUbuntu1604DistroNodes returns true if any of the agent pools or masters are running the base Ubuntu 16.04-LTS image
func (p *Properties) HasUbuntu1604DistroNodes() bool {
	if len(p.AgentPoolProfiles) > 0 {
		for _, ap := range p.AgentPoolProfiles {
			if ap.Distro == Ubuntu {
				return true
			}
		}
	}
	if p.MasterProfile != nil {
		return p.MasterProfile.Distro == Ubuntu
	}
	return false
}

// HasUbuntu1804DistroNodes returns true if any of the agent pools or masters are running the base Ubuntu 18.04-LTS image
func (p *Properties) HasUbuntu1804DistroNodes() bool {
	if len(p.AgentPoolProfiles) > 0 {
		for _, ap := range p.AgentPoolProfiles {
			if ap.Distro == Ubuntu1804 {
				return true
			}
		}
	}
	if p.MasterProfile != nil {
		return p.MasterProfile.Distro == Ubuntu1804
	}
	return false
}

// HasAvailabilityZones returns true if the cluster contains a profile with zones
func (p *Properties) HasAvailabilityZones() bool {
	hasZones := p.MasterProfile != nil && p.MasterProfile.HasAvailabilityZones()
	if !hasZones && p.AgentPoolProfiles != nil {
		for _, agentPoolProfile := range p.AgentPoolProfiles {
			if agentPoolProfile.HasAvailabilityZones() {
				hasZones = true
				break
			}
		}
	}
	return hasZones
}

// GetNonMasqueradeCIDR returns the non-masquerade CIDR for the ip-masq-agent.
func (p *Properties) GetNonMasqueradeCIDR() string {
	var nonMasqCidr string
	if !p.IsHostedMasterProfile() {
		if p.OrchestratorProfile.IsAzureCNI() {
			if p.MasterProfile != nil && p.MasterProfile.IsCustomVNET() {
				nonMasqCidr = p.MasterProfile.VnetCidr
			} else {
				nonMasqCidr = DefaultVNETCIDR
			}
		} else {
			// kube-proxy still only understands single cidr and is not changed for ipv6 dual stack phase 1
			// so only pass the ipv4 cidr which is the first one in the list as arg to kube proxy
			if p.FeatureFlags.IsFeatureEnabled("EnableIPv6DualStack") {
				nonMasqCidr = strings.Split(p.OrchestratorProfile.KubernetesConfig.ClusterSubnet, ",")[0]
			} else {
				nonMasqCidr = p.OrchestratorProfile.KubernetesConfig.ClusterSubnet
			}
		}
	}
	return nonMasqCidr
}

// GetAzureCNICidr returns the default CNI Cidr if Azure CNI is enabled.
func (p *Properties) GetAzureCNICidr() string {
	var masqCNIIP string
	if p.OrchestratorProfile != nil && p.OrchestratorProfile.IsAzureCNI() {
		masqCNIIP = DefaultCNICIDR
	}
	return masqCNIIP
}

// GetMasterFQDN returns the master FQDN.
func (p *Properties) GetMasterFQDN() string {
	if p.IsHostedMasterProfile() {
		return p.HostedMasterProfile.FQDN
	}

	return p.MasterProfile.FQDN
}

// AnyAgentHasLoadBalancerBackendAddressPoolIDs returns true if any of the agent profiles contains LoadBalancerBackendAddressPoolIDs
func (p *Properties) AnyAgentHasLoadBalancerBackendAddressPoolIDs() bool {
	for _, agentPoolProfile := range p.AgentPoolProfiles {
		if agentPoolProfile.LoadBalancerBackendAddressPoolIDs != nil {
			return true
		}
	}
	return false
}

// HasImageRef returns true if the customer brought os image
func (m *MasterProfile) HasImageRef() bool {
	return m.ImageRef != nil && len(m.ImageRef.Name) > 0 && len(m.ImageRef.ResourceGroup) > 0
}

// HasImageGallery returns true if the customer brought os image from Shared Image Gallery
func (m *MasterProfile) HasImageGallery() bool {
	return m.ImageRef != nil && len(m.ImageRef.SubscriptionID) > 0 && len(m.ImageRef.Gallery) > 0 && len(m.ImageRef.Version) > 0
}

// IsCustomVNET returns true if the customer brought their own VNET
func (m *MasterProfile) IsCustomVNET() bool {
	return len(m.VnetSubnetID) > 0
}

// IsManagedDisks returns true if the master specified managed disks
func (m *MasterProfile) IsManagedDisks() bool {
	return m.StorageProfile == ManagedDisks
}

// IsStorageAccount returns true if the master specified storage account
func (m *MasterProfile) IsStorageAccount() bool {
	return m.StorageProfile == StorageAccount
}

// IsRHEL returns true if the master specified a RHEL distro
func (m *MasterProfile) IsRHEL() bool {
	return m.Distro == RHEL
}

// IsCoreOS returns true if the master specified a CoreOS distro
func (m *MasterProfile) IsCoreOS() bool {
	return m.Distro == CoreOS
}

// IsVHDDistro returns true if the distro uses VHD SKUs
func (m *MasterProfile) IsVHDDistro() bool {
	return m.Distro == AKSUbuntu1604 || m.Distro == AKSUbuntu1804
}

// IsVirtualMachineScaleSets returns true if the master availability profile is VMSS
func (m *MasterProfile) IsVirtualMachineScaleSets() bool {
	return m.AvailabilityProfile == VirtualMachineScaleSets
}

// GetFirstConsecutiveStaticIPAddress returns the first static IP address of the given subnet.
func (m *MasterProfile) GetFirstConsecutiveStaticIPAddress(subnetStr string) string {
	_, subnet, err := net.ParseCIDR(subnetStr)
	if err != nil {
		return DefaultFirstConsecutiveKubernetesStaticIP
	}

	// Find the first and last octet of the host bits.
	ones, bits := subnet.Mask.Size()
	firstOctet := ones / 8
	lastOctet := bits/8 - 1

	if m.IsVirtualMachineScaleSets() {
		subnet.IP[lastOctet] = DefaultKubernetesFirstConsecutiveStaticIPOffsetVMSS
	} else {
		// Set the remaining host bits in the first octet.
		subnet.IP[firstOctet] |= (1 << byte((8 - (ones % 8)))) - 1

		// Fill the intermediate octets with 1s and last octet with offset. This is done so to match
		// the existing behavior of allocating static IP addresses from the last /24 of the subnet.
		for i := firstOctet + 1; i < lastOctet; i++ {
			subnet.IP[i] = 255
		}
		subnet.IP[lastOctet] = DefaultKubernetesFirstConsecutiveStaticIPOffset
	}

	return subnet.IP.String()
}

// HasAvailabilityZones returns true if the master profile has availability zones
func (m *MasterProfile) HasAvailabilityZones() bool {
	return m.AvailabilityZones != nil && len(m.AvailabilityZones) > 0
}

// IsUbuntu1604 returns true if the master profile distro is based on Ubuntu 16.04
func (m *MasterProfile) IsUbuntu1604() bool {
	switch m.Distro {
	case AKSUbuntu1604, Ubuntu, ACC1604:
		return true
	default:
		return false
	}
}

// IsUbuntu1804 returns true if the master profile distro is based on Ubuntu 18.04
func (m *MasterProfile) IsUbuntu1804() bool {
	switch m.Distro {
	case AKSUbuntu1804, Ubuntu1804:
		return true
	default:
		return false
	}
}

// IsUbuntu returns true if the master profile distro is any ubuntu distro
func (m *MasterProfile) IsUbuntu() bool {
	return m.IsUbuntu1604() || m.IsUbuntu1804()
}

// IsUbuntuNonVHD returns true if the distro uses a base Ubuntu image
func (m *MasterProfile) IsUbuntuNonVHD() bool {
	return m.IsUbuntu() && !m.IsVHDDistro()
}

// HasMultipleNodes returns true if there are more than one master nodes
func (m *MasterProfile) HasMultipleNodes() bool {
	return m.Count > 1
}

// HasCosmosEtcd returns true if cosmos etcd configuration is enabled
func (m *MasterProfile) HasCosmosEtcd() bool {
	return to.Bool(m.CosmosEtcd)
}

// GetCosmosEndPointURI returns the URI string for the cosmos etcd endpoint
func (m *MasterProfile) GetCosmosEndPointURI() string {
	if m.HasCosmosEtcd() {
		return fmt.Sprintf(etcdEndpointURIFmt, m.DNSPrefix)
	}
	return ""
}

// HasImageRef returns true if the customer brought os image
func (a *AgentPoolProfile) HasImageRef() bool {
	imageRef := a.ImageRef
	return imageRef != nil && len(imageRef.Name) > 0 && len(imageRef.ResourceGroup) > 0
}

// HasImageGallery returns true if the customer brought os image from Shared Image Gallery
func (a *AgentPoolProfile) HasImageGallery() bool {
	imageRef := a.ImageRef
	return imageRef != nil && len(imageRef.SubscriptionID) > 0 && len(imageRef.Gallery) > 0 && len(imageRef.Version) > 0
}

// IsCustomVNET returns true if the customer brought their own VNET
func (a *AgentPoolProfile) IsCustomVNET() bool {
	return len(a.VnetSubnetID) > 0
}

// IsWindows returns true if the agent pool is windows
func (a *AgentPoolProfile) IsWindows() bool {
	return a.OSType == Windows
}

// IsLinux returns true if the agent pool is linux
func (a *AgentPoolProfile) IsLinux() bool {
	return a.OSType == Linux
}

// IsRHEL returns true if the agent pool specified a RHEL distro
func (a *AgentPoolProfile) IsRHEL() bool {
	return a.OSType == Linux && a.Distro == RHEL
}

// IsCoreOS returns true if the agent specified a CoreOS distro
func (a *AgentPoolProfile) IsCoreOS() bool {
	return a.OSType == Linux && a.Distro == CoreOS
}

// IsVHDDistro returns true if the distro uses VHD SKUs
func (a *AgentPoolProfile) IsVHDDistro() bool {
	return a.Distro == AKSUbuntu1604 || a.Distro == AKSUbuntu1804
}

// IsAvailabilitySets returns true if the customer specified disks
func (a *AgentPoolProfile) IsAvailabilitySets() bool {
	return a.AvailabilityProfile == AvailabilitySet
}

// IsVirtualMachineScaleSets returns true if the agent pool availability profile is VMSS
func (a *AgentPoolProfile) IsVirtualMachineScaleSets() bool {
	return a.AvailabilityProfile == VirtualMachineScaleSets
}

// IsLowPriorityScaleSet returns true if the VMSS is Low Priority
func (a *AgentPoolProfile) IsLowPriorityScaleSet() bool {
	return a.AvailabilityProfile == VirtualMachineScaleSets && a.ScaleSetPriority == ScaleSetPriorityLow
}

// IsManagedDisks returns true if the customer specified disks
func (a *AgentPoolProfile) IsManagedDisks() bool {
	return a.StorageProfile == ManagedDisks
}

// IsStorageAccount returns true if the customer specified storage account
func (a *AgentPoolProfile) IsStorageAccount() bool {
	return a.StorageProfile == StorageAccount
}

// IsStorageAccount returns true if the customer specified ephemeral disks
func (a *AgentPoolProfile) IsEphemeral() bool {
	return a.StorageProfile == Ephemeral
}

// HasDisks returns true if the customer specified disks
func (a *AgentPoolProfile) HasDisks() bool {
	return len(a.DiskSizesGB) > 0
}

// HasAvailabilityZones returns true if the agent pool has availability zones
func (a *AgentPoolProfile) HasAvailabilityZones() bool {
	return a.AvailabilityZones != nil && len(a.AvailabilityZones) > 0
}

// IsUbuntu1604 returns true if the agent pool profile distro is based on Ubuntu 16.04
func (a *AgentPoolProfile) IsUbuntu1604() bool {
	if a.OSType != Windows {
		switch a.Distro {
		case AKSUbuntu1604, Ubuntu, ACC1604:
			return true
		default:
			return false
		}
	}
	return false
}

// IsUbuntu1804 returns true if the agent pool profile distro is based on Ubuntu 16.04
func (a *AgentPoolProfile) IsUbuntu1804() bool {
	if a.OSType != Windows {
		switch a.Distro {
		case AKSUbuntu1804, Ubuntu1804:
			return true
		default:
			return false
		}
	}
	return false
}

// IsUbuntu returns true if the master profile distro is any ubuntu distro
func (a *AgentPoolProfile) IsUbuntu() bool {
	return a.IsUbuntu1604() || a.IsUbuntu1804()
}

// IsUbuntuNonVHD returns true if the distro uses a base Ubuntu image
func (a *AgentPoolProfile) IsUbuntuNonVHD() bool {
	return a.IsUbuntu() && !a.IsVHDDistro()
}

// GetKubernetesLabels returns a k8s API-compliant labels string for nodes in this profile
func (a *AgentPoolProfile) GetKubernetesLabels(rg string, deprecated bool) string {
	var buf bytes.Buffer
	buf.WriteString("kubernetes.azure.com/role=agent")
	if deprecated {
		buf.WriteString(",node-role.kubernetes.io/agent=")
		buf.WriteString(",kubernetes.io/role=agent")
	}
	buf.WriteString(fmt.Sprintf(",agentpool=%s", a.Name))
	if a.StorageProfile == ManagedDisks {
		storagetier, _ := common.GetStorageAccountType(a.VMSize)
		buf.WriteString(fmt.Sprintf(",storageprofile=managed,storagetier=%s", storagetier))
	}
	if common.IsNvidiaEnabledSKU(a.VMSize) {
		accelerator := "nvidia"
		buf.WriteString(fmt.Sprintf(",accelerator=%s", accelerator))
	}
	buf.WriteString(fmt.Sprintf(",kubernetes.azure.com/cluster=%s", rg))
	keys := []string{}
	for key := range a.CustomNodeLabels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		buf.WriteString(fmt.Sprintf(",%s=%s", key, a.CustomNodeLabels[key]))
	}
	return buf.String()
}

// HasSecrets returns true if the customer specified secrets to install
func (w *WindowsProfile) HasSecrets() bool {
	return len(w.Secrets) > 0
}

// HasCustomImage returns true if there is a custom windows os image url specified
func (w *WindowsProfile) HasCustomImage() bool {
	return len(w.WindowsImageSourceURL) > 0
}

// GetWindowsDockerVersion gets the docker version specified or returns default value
func (w *WindowsProfile) GetWindowsDockerVersion() string {
	if w.WindowsDockerVersion != "" {
		return w.WindowsDockerVersion
	}
	return KubernetesWindowsDockerVersion
}

// GetWindowsSku gets the marketplace sku specified (such as Datacenter-Core-1809-with-Containers-smalldisk) or returns default value
func (w *WindowsProfile) GetWindowsSku() string {
	if w.WindowsSku != "" {
		return w.WindowsSku
	}
	return KubernetesDefaultWindowsSku
}

// GetEnableWindowsUpdate gets the flag for enable windows update or returns the default value
func (w *WindowsProfile) GetEnableWindowsUpdate() bool {
	if w.EnableAutomaticUpdates != nil {
		return *w.EnableAutomaticUpdates
	}
	return DefaultEnableAutomaticUpdates
}

// HasSecrets returns true if the customer specified secrets to install
func (l *LinuxProfile) HasSecrets() bool {
	return len(l.Secrets) > 0
}

// HasSearchDomain returns true if the customer specified secrets to install
func (l *LinuxProfile) HasSearchDomain() bool {
	if l.CustomSearchDomain != nil {
		if l.CustomSearchDomain.Name != "" && l.CustomSearchDomain.RealmPassword != "" && l.CustomSearchDomain.RealmUser != "" {
			return true
		}
	}
	return false
}

// HasCustomNodesDNS returns true if the customer specified a dns server
func (l *LinuxProfile) HasCustomNodesDNS() bool {
	if l.CustomNodesDNS != nil {
		if l.CustomNodesDNS.DNSServer != "" {
			return true
		}
	}
	return false
}

// IsSwarmMode returns true if this template is for Swarm Mode orchestrator
func (o *OrchestratorProfile) IsSwarmMode() bool {
	return o.OrchestratorType == SwarmMode
}

// IsKubernetes returns true if this template is for Kubernetes orchestrator
func (o *OrchestratorProfile) IsKubernetes() bool {
	return o.OrchestratorType == Kubernetes
}

// IsDCOS returns true if this template is for DCOS orchestrator
func (o *OrchestratorProfile) IsDCOS() bool {
	return o.OrchestratorType == DCOS
}

// IsDCOS19 returns true if this is a DCOS 1.9 orchestrator using the latest version
func (o *OrchestratorProfile) IsDCOS19() bool {
	return o.OrchestratorType == DCOS &&
		(o.OrchestratorVersion == common.DCOSVersion1Dot9Dot0 ||
			o.OrchestratorVersion == common.DCOSVersion1Dot9Dot8)
}

// IsAzureCNI returns true if Azure CNI network plugin is enabled
func (o *OrchestratorProfile) IsAzureCNI() bool {
	if o.KubernetesConfig != nil {
		return o.KubernetesConfig.NetworkPlugin == NetworkPluginAzure
	}
	return false
}

// RequireRouteTable returns true if this deployment requires routing table
func (o *OrchestratorProfile) RequireRouteTable() bool {
	switch o.OrchestratorType {
	case Kubernetes:
		if o.IsAzureCNI() || NetworkPolicyCilium == o.KubernetesConfig.NetworkPolicy || "flannel" == o.KubernetesConfig.NetworkPlugin {
			return false
		}
		return true
	default:
		return false
	}
}

// IsPrivateCluster returns true if this deployment is a private cluster
func (o *OrchestratorProfile) IsPrivateCluster() bool {
	if !o.IsKubernetes() {
		return false
	}
	return o.KubernetesConfig != nil && o.KubernetesConfig.PrivateCluster != nil && to.Bool(o.KubernetesConfig.PrivateCluster.Enabled)
}

// NeedsExecHealthz returns whether or not we have a configuration that requires exechealthz pod anywhere
func (o *OrchestratorProfile) NeedsExecHealthz() bool {
	return o.IsKubernetes() &&
		common.IsKubernetesVersionGe(o.OrchestratorVersion, "1.7.0") &&
		!common.IsKubernetesVersionGe(o.OrchestratorVersion, "1.9.0")
}

// HasAadProfile  returns true if the has aad profile
func (p *Properties) HasAadProfile() bool {
	return p.AADProfile != nil
}

// GetAPIServerEtcdAPIVersion Used to set apiserver's etcdapi version
func (o *OrchestratorProfile) GetAPIServerEtcdAPIVersion() string {
	if o.KubernetesConfig != nil {
		// if we are here, version has already been validated..
		etcdVersion, _ := semver.Make(o.KubernetesConfig.EtcdVersion)
		return "etcd" + strconv.FormatUint(etcdVersion.Major, 10)
	}
	return ""
}

// GetAddonByName returns the KubernetesAddon instance with name `addonName`
func (k *KubernetesConfig) GetAddonByName(addonName string) KubernetesAddon {
	var kubeAddon KubernetesAddon
	for _, addon := range k.Addons {
		if addon.Name == addonName {
			kubeAddon = addon
			break
		}
	}
	return kubeAddon
}

// GetAddonScript retrieves the raw script data specified as input for the k8s addon with name "addonName".
func (k *KubernetesConfig) GetAddonScript(addonName string) string {
	kubeAddon := k.GetAddonByName(addonName)
	return kubeAddon.Data
}

// IsAddonEnabled checks whether a k8s addon with name "addonName" is enabled or not based on the Enabled field of KubernetesAddon.
// If the value of Enabled is nil, the "defaultValue" is returned.
func (k *KubernetesConfig) IsAddonEnabled(addonName string) bool {
	kubeAddon := k.GetAddonByName(addonName)
	return kubeAddon.IsEnabled()
}

// IsAADPodIdentityEnabled checks if the AAD pod identity addon is enabled
func (k *KubernetesConfig) IsAADPodIdentityEnabled() bool {
	return k.IsAddonEnabled(AADPodIdentityAddonName)
}

// IsClusterAutoscalerEnabled checks if the cluster autoscaler addon is enabled
func (k *KubernetesConfig) IsClusterAutoscalerEnabled() bool {
	return k.IsAddonEnabled(ClusterAutoscalerAddonName)
}

// IsAppGWIngressEnabled checks if the appgw ingress addon is enabled
func (k *KubernetesConfig) IsAppGWIngressEnabled() bool {
	return k.IsAddonEnabled(AppGwIngressAddonName)
}

// IsIPMasqAgentEnabled checks if the ip-masq-agent addon is enabled
func (k *KubernetesConfig) IsIPMasqAgentEnabled() bool {
	return k.IsAddonEnabled(IPMASQAgentAddonName)
}

// IsRBACEnabled checks if RBAC is enabled
func (k *KubernetesConfig) IsRBACEnabled() bool {
	if k.EnableRbac != nil {
		return to.Bool(k.EnableRbac)
	}
	return false
}

// UserAssignedIDEnabled checks if the user assigned ID is enabled or not.
func (k *KubernetesConfig) UserAssignedIDEnabled() bool {
	return k.UseManagedIdentity && k.UserAssignedID != ""
}

// UserAssignedClientIDEnabled checks if the user assigned client ID is enabled or not.
func (k *KubernetesConfig) UserAssignedClientIDEnabled() bool {
	return k.UseManagedIdentity && k.UserAssignedClientID != ""
}

// GetUserAssignedID returns the user assigned ID if it is enabled.
func (k *KubernetesConfig) GetUserAssignedID() string {
	if k.UserAssignedIDEnabled() {
		return k.UserAssignedID
	}
	return ""
}

// GetUserAssignedClientID returns the user assigned client ID if it is enabled.
func (k *KubernetesConfig) GetUserAssignedClientID() string {
	if k.UserAssignedClientIDEnabled() {
		return k.UserAssignedClientID
	}
	return ""
}

// GetOrderedKubeletConfigString returns an ordered string of key/val pairs
func (k *KubernetesConfig) GetOrderedKubeletConfigString() string {
	keys := []string{}
	for key := range k.KubeletConfig {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	for _, key := range keys {
		buf.WriteString(fmt.Sprintf("%s=%s ", key, k.KubeletConfig[key]))
	}
	return buf.String()
}

// GetOrderedKubeletConfigStringForPowershell returns an ordered string of key/val pairs for Powershell script consumption
func (k *KubernetesConfig) GetOrderedKubeletConfigStringForPowershell() string {
	keys := []string{}
	for key := range k.KubeletConfig {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	for _, key := range keys {
		buf.WriteString(fmt.Sprintf("\"%s=%s\", ", key, k.KubeletConfig[key]))
	}
	return strings.TrimSuffix(buf.String(), ", ")
}

// IsNSeriesSKU returns true if the agent pool contains an N-series (NVIDIA GPU) VM
func (a *AgentPoolProfile) IsNSeriesSKU() bool {
	return common.IsNvidiaEnabledSKU(a.VMSize)
}

// HasNSeriesSKU returns whether or not there is an N series SKU agent pool
func (p *Properties) HasNSeriesSKU() bool {
	for _, profile := range p.AgentPoolProfiles {
		if strings.Contains(profile.VMSize, "Standard_N") {
			return true
		}
	}
	return false
}

// IsNVIDIADevicePluginEnabled checks if the NVIDIA Device Plugin addon is enabled
// It is enabled by default if agents contain a GPU and Kubernetes version is >= 1.10.0
func (p *Properties) IsNVIDIADevicePluginEnabled() bool {
	if p.OrchestratorProfile == nil || p.OrchestratorProfile.KubernetesConfig == nil {
		return false
	}
	return p.OrchestratorProfile.KubernetesConfig.IsAddonEnabled(NVIDIADevicePluginAddonName)
}

// IsAzureStackCloud return true if the cloud is AzureStack
func (p *Properties) IsAzureStackCloud() bool {
	return p.CustomCloudProfile != nil
}

// GetCustomEnvironmentJSON return the JSON format string for custom environment
func (p *Properties) GetCustomEnvironmentJSON(escape bool) (string, error) {
	var environmentJSON string
	if p.IsAzureStackCloud() {
		bytes, err := json.Marshal(p.CustomCloudProfile.Environment)
		if err != nil {
			return "", fmt.Errorf("Could not serialize Environment object - %s", err.Error())
		}
		environmentJSON = string(bytes)
		if escape {
			environmentJSON = strings.Replace(environmentJSON, "\"", "\\\"", -1)
		}
	}
	return environmentJSON, nil
}

// GetCustomCloudName returns name of environment if customCloudProfile is provided, returns empty string if customCloudProfile is empty.
// Because customCloudProfile is empty for deployment is AzurePublicCloud, AzureChinaCloud,AzureGermanCloud,AzureUSGovernmentCloud,
// the return value will be empty string for those clouds
func (p *Properties) GetCustomCloudName() string {
	var cloudProfileName string
	if p.IsAzureStackCloud() {
		cloudProfileName = p.CustomCloudProfile.Environment.Name
	}
	return cloudProfileName
}

// GetLocations returns all supported regions.
// If AzureStackCloud, GetLocations provides the location of container service
// If AzurePublicCloud, AzureChinaCloud,AzureGermanCloud or AzureUSGovernmentCloud, GetLocations provides all azure regions in prod.
func (cs *ContainerService) GetLocations() []string {
	var allLocations []string
	if cs.Properties.IsAzureStackCloud() {
		allLocations = []string{cs.Location}
	} else {
		allLocations = helpers.GetAzureLocations()
	}
	return allLocations
}

// GetCustomCloudAuthenticationMethod returns authentication method which k8s azure cloud provider will use
// For AzurePublicCloud,AzureChinaCloud,azureGermanCloud,AzureUSGovernmentCloud, it will be always be client_secret
// For AzureStackCloud, if it is specified in configuration, the value will be used, if not ,the default value is client_secret.
func (p *Properties) GetCustomCloudAuthenticationMethod() string {
	if p.IsAzureStackCloud() {
		return p.CustomCloudProfile.AuthenticationMethod
	}
	return ClientSecretAuthMethod
}

// GetCustomCloudIdentitySystem returns identity system method for azure stack.
// For AzurePublicCloud,AzureChinaCloud,azureGermanCloud,AzureUSGovernmentCloud, it will be always be AzureAD
// For AzureStackCloud, if it is specified in configuration, the value will be used, if not ,the default value is AzureAD.
func (p *Properties) GetCustomCloudIdentitySystem() string {
	if p.IsAzureStackCloud() {
		return p.CustomCloudProfile.IdentitySystem
	}
	return AzureADIdentitySystem
}

// IsNvidiaDevicePluginCapable determines if the cluster definition is compatible with the nvidia-device-plugin daemonset
func (p *Properties) IsNvidiaDevicePluginCapable() bool {
	return p.HasNSeriesSKU() && common.IsKubernetesVersionGe(p.OrchestratorProfile.OrchestratorVersion, "1.10.0")
}

// SetCloudProviderRateLimitDefaults sets default cloudprovider rate limiter config
func (p *Properties) SetCloudProviderRateLimitDefaults() {
	if p.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitBucket == 0 {
		if p.HasVMSSAgentPool() && !p.IsHostedMasterProfile() {
			var rateLimitBucket int
			for _, profile := range p.AgentPoolProfiles {
				if profile.AvailabilityProfile == VirtualMachineScaleSets {
					rateLimitBucket += common.MaxAgentCount
				}

			}
			p.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitBucket = rateLimitBucket
		} else {
			p.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitBucket = DefaultKubernetesCloudProviderRateLimitBucket
		}
	}
	if p.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitQPS == 0 {
		if (DefaultKubernetesCloudProviderRateLimitQPS / float64(p.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitBucket)) < common.MinCloudProviderQPSToBucketFactor {
			p.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitQPS = float64(p.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitBucket) * common.MinCloudProviderQPSToBucketFactor
		} else {
			p.OrchestratorProfile.KubernetesConfig.CloudProviderRateLimitQPS = DefaultKubernetesCloudProviderRateLimitQPS
		}
	}
}

// IsReschedulerEnabled checks if the rescheduler addon is enabled
func (k *KubernetesConfig) IsReschedulerEnabled() bool {
	return k.IsAddonEnabled(ReschedulerAddonName)
}

// PrivateJumpboxProvision checks if a private cluster has jumpbox auto-provisioning
func (k *KubernetesConfig) PrivateJumpboxProvision() bool {
	if k != nil && k.PrivateCluster != nil && *k.PrivateCluster.Enabled && k.PrivateCluster.JumpboxProfile != nil {
		return true
	}
	return false
}

// RequiresDocker returns if the kubernetes settings require docker binary to be installed.
func (k *KubernetesConfig) RequiresDocker() bool {
	runtime := strings.ToLower(k.ContainerRuntime)
	return runtime == Docker || runtime == ""
}

// SetCloudProviderBackoffDefaults sets default cloudprovider backoff config
func (k *KubernetesConfig) SetCloudProviderBackoffDefaults() {
	if k.CloudProviderBackoffDuration == 0 {
		k.CloudProviderBackoffDuration = DefaultKubernetesCloudProviderBackoffDuration
	}
	if k.CloudProviderBackoffExponent == 0 {
		k.CloudProviderBackoffExponent = DefaultKubernetesCloudProviderBackoffExponent
	}
	if k.CloudProviderBackoffJitter == 0 {
		k.CloudProviderBackoffJitter = DefaultKubernetesCloudProviderBackoffJitter
	}
	if k.CloudProviderBackoffRetries == 0 {
		k.CloudProviderBackoffRetries = DefaultKubernetesCloudProviderBackoffRetries
	}
}

// GetAzureCNIURLLinux returns the full URL to source Azure CNI binaries from
func (k *KubernetesConfig) GetAzureCNIURLLinux(cloudSpecConfig AzureEnvironmentSpecConfig) string {
	if k.AzureCNIURLLinux != "" {
		return k.AzureCNIURLLinux
	}
	return cloudSpecConfig.KubernetesSpecConfig.VnetCNILinuxPluginsDownloadURL
}

// GetAzureCNIURLWindows returns the full URL to source Azure CNI binaries from
func (k *KubernetesConfig) GetAzureCNIURLWindows(cloudSpecConfig AzureEnvironmentSpecConfig) string {
	if k.AzureCNIURLWindows != "" {
		return k.AzureCNIURLWindows
	}
	return cloudSpecConfig.KubernetesSpecConfig.VnetCNIWindowsPluginsDownloadURL
}

// IsFeatureEnabled returns true if a feature flag is on for the provided feature
func (f *FeatureFlags) IsFeatureEnabled(feature string) bool {
	if f != nil {
		switch feature {
		case "CSERunInBackground":
			return f.EnableCSERunInBackground
		case "BlockOutboundInternet":
			return f.BlockOutboundInternet
		case "EnableIPv6DualStack":
			return f.EnableIPv6DualStack
		default:
			return false
		}
	}
	return false
}

// GetCloudSpecConfig returns the Kubernetes container images URL configurations based on the deploy target environment.
//for example: if the target is the public azure, then the default container image url should be k8s.gcr.io/...
//if the target is azure china, then the default container image should be mirror.azure.cn:5000/google_container/...
func (cs *ContainerService) GetCloudSpecConfig() AzureEnvironmentSpecConfig {
	targetEnv := helpers.GetTargetEnv(cs.Location, cs.Properties.GetCustomCloudName())
	return AzureCloudSpecEnvMap[targetEnv]
}

// IsAKSBillingEnabled checks if the AKS Billing Extension should be enabled for a cloud environment.
func (cs *ContainerService) IsAKSBillingEnabled() bool {
	cloudSpecConfig := cs.GetCloudSpecConfig()
	return cloudSpecConfig.CloudName == AzurePublicCloud || cloudSpecConfig.CloudName == AzureChinaCloud || cloudSpecConfig.CloudName == AzureUSGovernmentCloud
}

// GetAzureProdFQDN returns the formatted FQDN string for a given apimodel.
func (cs *ContainerService) GetAzureProdFQDN() string {
	return FormatProdFQDNByLocation(cs.Properties.MasterProfile.DNSPrefix, cs.Location, cs.Properties.GetCustomCloudName())
}

// SetPlatformFaultDomainCount sets the fault domain count value for all VMASes in a cluster.
func (cs *ContainerService) SetPlatformFaultDomainCount(count int) {
	// Assume that all VMASes in the cluster share a value for platformFaultDomainCount
	if cs.Properties.MasterProfile != nil {
		cs.Properties.MasterProfile.PlatformFaultDomainCount = &count
	}
	for _, pool := range cs.Properties.AgentPoolProfiles {
		pool.PlatformFaultDomainCount = &count
	}
}

// FormatAzureProdFQDNByLocation constructs an Azure prod fqdn
func FormatAzureProdFQDNByLocation(fqdnPrefix string, location string) string {
	targetEnv := helpers.GetCloudTargetEnv(location)
	FQDNFormat := AzureCloudSpecEnvMap[targetEnv].EndpointConfig.ResourceManagerVMDNSSuffix
	return fmt.Sprintf("%s.%s."+FQDNFormat, fqdnPrefix, location)
}

// FormatProdFQDNByLocation constructs an Azure prod fqdn with custom cloud profile
// CustomCloudName is name of environment if customCloudProfile is provided, it will be empty string if customCloudProfile is empty.
// Because customCloudProfile is empty for deployment for AzurePublicCloud, AzureChinaCloud,AzureGermanCloud,AzureUSGovernmentCloud,
// The customCloudName value will be empty string for those clouds
func FormatProdFQDNByLocation(fqdnPrefix string, location string, cloudName string) string {
	targetEnv := helpers.GetTargetEnv(location, cloudName)
	FQDNFormat := AzureCloudSpecEnvMap[targetEnv].EndpointConfig.ResourceManagerVMDNSSuffix
	return fmt.Sprintf("%s.%s."+FQDNFormat, fqdnPrefix, location)
}
