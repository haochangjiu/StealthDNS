//go:build darwin

package handler

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/OpenNHP/StealthDNS/common"
	"github.com/OpenNHP/opennhp/nhp/log"
)

type MacHandler struct {
	isRoot        bool
	dnsSetupFlag  bool
	interfaceName string
	backupDNS     []string
	upstreamDNS   string
	dhcpMode      bool // Whether DNS was originally from DHCP
}

func NewDNSHandler() *MacHandler {
	h := &MacHandler{
		dnsSetupFlag: false,
	}
	h.isRootPermission()
	return h
}

func (h *MacHandler) SetStealthDNS() (bool, error) {
	if !h.isRoot {
		log.Warning("The current account does not have root privileges. Please manually configure the alternate DNS.")
		return false, nil
	}
	err := h.getInterfaceName()
	if err != nil {
		log.Error("Failed to retrieve network settings: %v", err)
		return false, err
	}
	err = h.backupAndSet()
	if err != nil {
		return false, err
	}
	h.dnsSetupFlag = true
	return true, nil
}

func (h *MacHandler) RemoveStealthDNS() {
	if !h.dnsSetupFlag {
		log.Warning("The DNS proxy address is configured manually, and the StealthDNS service does not automatically restore the settings.")
		return
	}

	if !h.isRoot {
		log.Warning("The current account does not have root privileges. Please manually configure the alternate DNS.")
		return
	}

	if len(h.interfaceName) == 0 {
		log.Warning("No interfaces found, DNS reset is not required.")
		return
	}

	// If DNS was originally from DHCP, restore to DHCP mode
	if h.dhcpMode {
		log.Info("Restoring DNS to DHCP mode for interface [%s]", h.interfaceName)
		cmd := exec.Command("networksetup", "-setdnsservers", h.interfaceName, "Empty")
		if err := cmd.Run(); err != nil {
			log.Error("dns network [%s] restore to DHCP fail: %v", h.interfaceName, err)
		} else {
			log.Info("DNS restored to DHCP mode successfully")
		}
		return
	}

	// Otherwise, restore to the backed up static DNS servers
	err := h.setDNS(h.backupDNS)
	if err != nil {
		log.Error("dns network [%s] reset fail: %v", h.interfaceName, err)
	}
}

func (h *MacHandler) getInterfaceName() error {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		service := strings.TrimSpace(scanner.Text())
		if service == "" || strings.HasPrefix(service, "*") {
			continue
		}
		if h.isNetworkServiceActive(service) {
			h.interfaceName = service
		}
	}
	if len(h.interfaceName) == 0 {
		return errors.New("no network service found")
	}
	return nil
}

func (h *MacHandler) isNetworkServiceActive(service string) bool {
	cmd := exec.Command("networksetup", "-getinfo", service)
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "IP address:") && !strings.Contains(string(output), "IP address: 0.0.0.0")
}

func (h *MacHandler) backupAndSet() error {
	// Try to store backup DNS, but don't fail if no DNS is set
	_ = h.storeBackupDNS()

	// Build DNS list with StealthDNS IP first
	dnsServices := make([]string, 0)
	dnsServices = append(dnsServices, common.StealthDnsIp)

	// Add backup DNS servers, filtering out empty or invalid IPs
	for _, dns := range h.backupDNS {
		dns = strings.TrimSpace(dns)
		if dns != "" && h.isValidIP(dns) {
			dnsServices = append(dnsServices, dns)
		}
	}

	// If no valid backup DNS, add default upstream DNS
	if len(dnsServices) == 1 {
		dnsServices = append(dnsServices, common.DefaultUpstreamDNS)
	}

	err := h.setDNS(dnsServices)
	return err
}

func (h *MacHandler) setDNS(dnsServers []string) error {
	// Filter out empty or invalid IP addresses
	validDNS := make([]string, 0)
	for _, dns := range dnsServers {
		dns = strings.TrimSpace(dns)
		if dns != "" && h.isValidIP(dns) {
			validDNS = append(validDNS, dns)
		}
	}

	if len(validDNS) == 0 {
		return fmt.Errorf("no valid DNS servers to set")
	}

	args := []string{"-setdnsservers", h.interfaceName}
	args = append(args, validDNS...)

	cmd := exec.Command("networksetup", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set DNS fail: %v, output: %s", err, string(output))
	}
	return nil
}

func (h *MacHandler) storeBackupDNS() error {
	// First try networksetup (only gets manually configured DNS)
	cmd := exec.Command("networksetup", "-getdnsservers", h.interfaceName)
	output, err := cmd.CombinedOutput()
	outputStr := strings.TrimSpace(string(output))

	var dnsServers []string
	var wasDHCP bool

	// Check if networksetup returned an error or empty result
	if err != nil || strings.Contains(outputStr, "aren't any DNS Servers set") {
		// networksetup doesn't show DHCP-assigned DNS, this means DNS is from DHCP
		wasDHCP = true
		log.Debug("networksetup returned no DNS, DNS is from DHCP. Trying scutil to get actual DNS servers...")
		dnsServers = h.getDNSServersFromScutil()
	} else {
		// Parse DNS servers from networksetup output (manually configured DNS)
		wasDHCP = false
		scanner := bufio.NewScanner(strings.NewReader(outputStr))
		for scanner.Scan() {
			dns := strings.TrimSpace(scanner.Text())
			// Skip empty lines and error messages
			if dns == "" ||
				strings.Contains(dns, "aren't any DNS Servers set") ||
				strings.Contains(dns, "is not a valid IP address") ||
				strings.Contains(dns, "No changes were saved") {
				continue
			}
			// Only add valid IP addresses that are not the StealthDNS IP
			if !strings.EqualFold(dns, common.StealthDnsIp) && h.isValidIP(dns) {
				dnsServers = append(dnsServers, dns)
			}
		}

		// If networksetup returned empty, try scutil as fallback (might be DHCP)
		if len(dnsServers) == 0 {
			log.Debug("networksetup returned empty DNS list, trying scutil to get actual DNS servers (might be DHCP)...")
			scutilDNS := h.getDNSServersFromScutil()
			if len(scutilDNS) > 0 {
				// If scutil found DNS but networksetup didn't, it's DHCP
				wasDHCP = true
				dnsServers = scutilDNS
			}
		}
	}

	// Record whether DNS was from DHCP
	h.dhcpMode = wasDHCP

	h.backupDNS = dnsServers
	if len(dnsServers) == 0 {
		log.Warning("Failed to obtain upstream DNS; using default DNS [%s] as the upstream DNS", common.DefaultUpstreamDNS)
		h.upstreamDNS = common.DefaultUpstreamDNS
	} else {
		h.upstreamDNS = dnsServers[0]
		if wasDHCP {
			log.Info("Obtained upstream DNS from DHCP: %s (from %d DNS servers)", h.upstreamDNS, len(dnsServers))
		} else {
			log.Info("Obtained upstream DNS from static config: %s (from %d DNS servers)", h.upstreamDNS, len(dnsServers))
		}
	}
	return nil
}

// getDNSServersFromScutil gets DNS servers using scutil (includes DHCP-assigned DNS)
func (h *MacHandler) getDNSServersFromScutil() []string {
	cmd := exec.Command("scutil", "--dns")
	output, err := cmd.Output()
	if err != nil {
		log.Debug("Failed to get DNS from scutil: %v", err)
		return nil
	}

	var dnsServers []string
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Look for lines like: nameserver[0] : 192.168.1.1
		if strings.Contains(line, "nameserver[") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				dns := strings.TrimSpace(parts[1])
				if dns != "" && h.isValidIP(dns) && !strings.EqualFold(dns, common.StealthDnsIp) {
					// Avoid duplicates
					duplicate := false
					for _, existing := range dnsServers {
						if existing == dns {
							duplicate = true
							break
						}
					}
					if !duplicate {
						dnsServers = append(dnsServers, dns)
					}
				}
			}
		}
	}

	return dnsServers
}

func (h *MacHandler) isRootPermission() {
	h.isRoot = os.Geteuid() == 0
}

func (h *MacHandler) GetUpstreamDNS() string {
	// Ensure we always return a valid IP address
	if h.upstreamDNS == "" || !h.isValidIP(h.upstreamDNS) {
		log.Warning("Invalid upstream DNS [%s], using default DNS [%s]", h.upstreamDNS, common.DefaultUpstreamDNS)
		return common.DefaultUpstreamDNS
	}
	return h.upstreamDNS
}

func (h *MacHandler) isValidIP(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.To4() != nil // Only accept IPv4 addresses
}
