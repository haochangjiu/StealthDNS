// Package nhpcore provides NHP (Network-Hiding Protocol) core functionality for mobile platforms.
// This package is designed to be compiled with gomobile for use in Android and iOS applications.
//
// Build for Android: gomobile bind -target=android -o nhpcore.aar .
// Build for iOS: gomobile bind -target=ios -o Nhpcore.xcframework .
package nhpcore

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/OpenNHP/opennhp/endpoints/agent"
	"github.com/OpenNHP/opennhp/nhp/common"
	_ "golang.org/x/mobile/bind"
)

// Constants
const (
	NhpDomainSuffix    = ".nhp"
	DefaultUpstreamDNS = "8.8.8.8"
	DefaultServerPort  = 62206
	Version            = "1.0.0"
)

// Global NHP agent instance
var (
	gAgentInstance *agent.UdpAgent
	gWorkDir       string
	gLogLevel      int
	gInitMutex     sync.Mutex
	gInitialized   bool

	// Resource mapping: resourceId -> KnockResource
	gResourceMap     map[string]*agent.KnockTarget
	gResourceMapLock sync.RWMutex
)

// InitializeWithConfig initializes the NHP agent with JSON config
// configJSON: JSON string containing agent config, servers, and resources
func InitializeWithConfig(workDir string, logLevel int, configJSON string) error {
	gInitMutex.Lock()
	defer gInitMutex.Unlock()

	if gInitialized {
		return nil
	}

	gWorkDir = workDir
	gLogLevel = logLevel
	gResourceMap = make(map[string]*agent.KnockTarget)

	// Parse config
	var config struct {
		Agent struct {
			PrivateKeyBase64 string `json:"privateKeyBase64"`
			UserId           string `json:"userId"`
			OrganizationId   string `json:"organizationId"`
			CipherScheme     int    `json:"cipherScheme"`
		} `json:"agent"`
		Servers []struct {
			Hostname     string `json:"hostname"`
			Ip           string `json:"ip"`
			Port         int    `json:"port"`
			PubKeyBase64 string `json:"pubKeyBase64"`
			ExpireTime   int64  `json:"expireTime"`
		} `json:"servers"`
		Resources []struct {
			AuthServiceId  string `json:"authServiceId"`
			ResourceId     string `json:"resourceId"`
			ServerIp       string `json:"serverIp"`
			ServerHostname string `json:"serverHostname"`
			ServerPort     int    `json:"serverPort"`
		} `json:"resources"`
	}

	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return err
	}

	// Create agent instance
	gAgentInstance = &agent.UdpAgent{}
	err := gAgentInstance.Start(workDir, logLevel)
	if err != nil {
		gAgentInstance = nil
		return err
	}

	// Add resources
	for _, res := range config.Resources {
		port := res.ServerPort
		if port == 0 {
			port = DefaultServerPort
		}
		resource := agent.KnockResource{
			AuthServiceId:  res.AuthServiceId,
			ResourceId:     res.ResourceId,
			ServerIp:       res.ServerIp,
			ServerHostname: res.ServerHostname,
			ServerPort:     port,
		}

		target := &agent.KnockTarget{
			KnockResource: resource,
			ServerPeer:    gAgentInstance.FindServerPeerFromResource(&resource),
		}

		gResourceMapLock.Lock()
		gResourceMap[res.ResourceId] = target
		gResourceMapLock.Unlock()

	}

	gInitialized = true
	return nil
}

// Cleanup releases all resources
func Cleanup() {
	gInitMutex.Lock()
	defer gInitMutex.Unlock()

	if gAgentInstance != nil {
		gAgentInstance.Stop()
		gAgentInstance = nil
	}

	gResourceMapLock.Lock()
	gResourceMap = nil
	gResourceMapLock.Unlock()

	gInitialized = false
}

// IsNHPDomain checks if the domain ends with .nhp suffix
func IsNHPDomain(domain string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSuffix(domain, ".")), NhpDomainSuffix)
}

// ExtractResourceID extracts the resource ID from an NHP domain
// e.g., "myapp.nhp" -> "myapp", "securitygroup.nhp" -> "securitygroup"
func ExtractResourceID(domain string) string {
	domain = strings.ToLower(strings.TrimSuffix(domain, "."))
	if idx := strings.Index(domain, NhpDomainSuffix); idx != -1 {
		return domain[:idx]
	}
	return domain
}

// GetKnockResultJSON performs knock and returns result as JSON
// resourceID: the resource ID extracted from domain (e.g., "demo" from "demo.nhp")
func GetKnockResultJSON(resourceID string) string {
	result := map[string]interface{}{
		"success":      false,
		"errorCode":    "",
		"errorMessage": "",
		"openTime":     0,
		"resourceHost": "",
	}

	if gAgentInstance == nil {
		result["errorCode"] = common.ErrNoAgentInstance.ErrorCode()
		result["errorMessage"] = "NHP agent not initialized"
		bytes, _ := json.Marshal(result)
		return string(bytes)
	}

	// Find resource by ID
	gResourceMapLock.RLock()
	target, found := gResourceMap[resourceID]
	gResourceMapLock.RUnlock()

	if !found {
		result["errorCode"] = "RESOURCE_NOT_FOUND"
		result["errorMessage"] = "Unknown resource: " + resourceID
		bytes, _ := json.Marshal(result)
		return string(bytes)
	}

	ackMsg, err := gAgentInstance.Knock(target)
	if err != nil {
		result["errorCode"] = ackMsg.ErrCode
		result["errorMessage"] = ackMsg.ErrMsg
		if result["errorMessage"] == "" {
			result["errorMessage"] = err.Error()
		}
		bytes, _ := json.Marshal(result)
		return string(bytes)
	}

	bytes, _ := json.Marshal(ackMsg)
	return string(bytes)
}

// IsInitialized returns whether the agent is initialized
func IsInitialized() bool {
	return gInitialized && gAgentInstance != nil
}
