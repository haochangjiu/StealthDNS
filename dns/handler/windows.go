//go:build windows

package handler

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenNHP/StealthDNS/common"
	"github.com/OpenNHP/opennhp/nhp/log"
	"golang.org/x/text/encoding/ianaindex"
	"golang.org/x/text/transform"
)

type WindowsHandler struct {
	isAdmin       bool
	dnsSetupFlag  bool
	interfaceName string
	backupDNS     []string
	upstreamDNS   string
	codePage      uint32
	dhcpEnabled   bool
}

type NetworkInterface struct {
	Index         string
	met           uint32
	status        string
	interfaceName string
	dnsAddress    []string
}

type NetworkAdapterConfig struct {
	InterfaceIndex       uint32   `wmi:"InterfaceIndex"`
	SettingID            string   `wmi:"SettingID"`
	DNSServerSearchOrder []string `wmi:"DNSServerSearchOrder"`
	IPEnabled            bool     `wmi:"IPEnabled"`
	DHCPEnabled          bool     `wmi:"DHCPEnabled"`
}

type WinInterInfo struct {
	Name   string
	Metric int
}

func NewDNSHandler() *WindowsHandler {
	h := &WindowsHandler{
		dnsSetupFlag: false,
	}
	h.isAdminPermission()
	h.getOEMCodePage()
	return h
}

func (h *WindowsHandler) GetUpstreamDNS() string {
	return h.upstreamDNS
}

func (h *WindowsHandler) SetStealthDNS() (bool, error) {
	if !h.isAdmin {
		log.Warning("The current account does not have administrator privileges. Please manually configure the alternate DNS.")
		return true, nil
	}
	networkInterfaces, err := h.getActiveInterface()
	if err != nil {
		log.Error("get active interface info fail: %v", err)
		return false, err
	}
	err = h.getPrimaryInterface(networkInterfaces)
	if err != nil {
		log.Error("get primary interface fail: %v", err)
		return false, err
	}
	err = h.setDNS(false)
	if err != nil {
		log.Error("set stealth dns proxy fail: %v", err)
		return false, err
	}
	h.dnsSetupFlag = true
	return true, nil
}
func (h *WindowsHandler) RemoveStealthDNS() {
	if !h.dnsSetupFlag {
		log.Warning("The DNS proxy address is configured manually, and the StealthDNS service does not automatically restore the settings.")
		return
	}
	if !h.isAdmin {
		log.Warning("The current account does not have administrator privileges. Please manually configure the alternate DNS.")
		return
	}
	err := h.setDNS(true)
	if err != nil {
		log.Error("restore the DNS configuration fail: %v", err)
		log.Debug("please manually restore the DNS configuration.")
	}
}

func (h *WindowsHandler) isAdminPermission() {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	if err == nil {
		h.isAdmin = true
	} else {
		h.isAdmin = false
	}
}

func (h *WindowsHandler) getOEMCodePage() {
	cmd := exec.Command("powershell", "-Command", "[Console]::OutputEncoding.CodePage")
	output, err := cmd.Output()
	if err != nil {
		h.codePage = 437
	}
	cp, err := strconv.ParseUint(strings.TrimSpace(string(output)), 10, 32)
	if err != nil {
		h.codePage = 437
	}
	h.codePage = uint32(cp)
	if h.codePage == 936 || h.codePage == 950 {
		cmd = exec.Command("chcp", "65001")
		cmd.Stdout = nil
		cmd.Stderr = nil
		if err = cmd.Run(); err == nil {
			h.codePage = 65001
		}
	}
}

func (h *WindowsHandler) decodeFromCodePage(data []byte) (string, error) {
	if h.codePage == 65001 {
		return string(data), nil
	}

	mimeName := fmt.Sprintf("cp%d", h.codePage)
	enc, err := ianaindex.MIME.Encoding(mimeName)
	if err != nil {
		return string(data), nil
	}

	decoder := enc.NewDecoder()
	utf8Str, _, err := transform.String(decoder, string(data))
	if err != nil {
		return string(data), nil
	}
	return utf8Str, nil
}

// Get the interface with the lowest metric (highest priority)
func (h *WindowsHandler) getPrimaryInterface(networkInterface []*NetworkInterface) error {
	for _, n := range networkInterface {
		isDNCP, address, err := h.getDNSInfo(n.interfaceName)
		if err == nil {
			h.dhcpEnabled = isDNCP
			h.backupDNS = address
			h.interfaceName = n.interfaceName
			return err
		}
	}
	return fmt.Errorf("no dns address found")
}

func (h *WindowsHandler) getDNSInfo(interfaceName string) (bool, []string, error) {
	dnsAddress := make([]string, 0)
	isDHCP := false
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "dns", interfaceName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, dnsAddress, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line, err := h.decodeFromCodePage(scanner.Bytes())
		if err != nil || len(line) == 0 {
			continue
		}
		ip := extractFirstIPv4(line)
		if ip != "" {
			dnsAddress = append(dnsAddress, ip)
		}
		if strings.Contains(line, "DHCP") && !isDHCP {
			isDHCP = true
		}
	}
	if len(dnsAddress) == 0 {
		return isDHCP, dnsAddress, fmt.Errorf("no dns address found")
	}
	return isDHCP, dnsAddress, nil
}

func extractFirstIPv4(s string) string {
	re := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	candidates := re.FindAllString(s, -1)

	for _, ip := range candidates {
		parts := strings.Split(ip, ".")
		if len(parts) != 4 {
			continue
		}

		valid := true
		for _, part := range parts {
			if len(part) > 1 && part[0] == '0' {
				valid = false
				break
			}
			num, err := strconv.Atoi(part)
			if err != nil || num < 0 || num > 255 {
				valid = false
				break
			}
		}

		if valid {
			return ip
		}
	}

	return "" // No valid IPv4 address detected
}

func (h *WindowsHandler) setDNS(restoreFlag bool) error {
	if restoreFlag && h.dhcpEnabled {
		cmd := exec.Command("netsh", "interface", "ipv4", "set", "dnsservers", h.interfaceName, "dhcp")
		return cmd.Run()
	}

	dnsList := make([]string, 0)
	if !restoreFlag {
		dnsList = append(dnsList, common.StealthDnsIp)
	}
	for _, dn := range h.backupDNS {
		if strings.EqualFold(dn, common.StealthDnsIp) && !restoreFlag {
			continue
		}
		dnsList = append(dnsList, dn)
	}
	cmd := exec.Command("netsh", "interface", "ip", "delete", "dns", h.interfaceName, "all")
	err := cmd.Run()
	if err != nil {
		log.Error("Failed to clear system DNS server configuration: %v", err)
		return err
	}
	cmd = exec.Command("netsh", "interface", "ipv4", "set", "dns",
		h.interfaceName, "static", dnsList[0], "primary")

	err = cmd.Run()

	if err != nil {
		log.Warning("Failed to set primary DNS：%v", err)
		return err
	}
	for i := 1; i < len(dnsList); i++ {
		cmd = exec.Command("netsh", "interface", "ipv4", "add", "dns",
			h.interfaceName, dnsList[i], "index="+strconv.Itoa(i+1))
		_ = cmd.Run() // Ignore failure to add secondary DNS.
	}

	return nil
}

func (h *WindowsHandler) getActiveInterface() ([]*NetworkInterface, error) {
	cmd := exec.Command("netsh", "interface", "ipv4", "show", "interfaces")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %v", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	headerSkipped := false

	networkInterfaces := make([]*NetworkInterface, 0)

	for scanner.Scan() {
		line, err := h.decodeFromCodePage(scanner.Bytes())
		if err != nil {
			continue
		}
		if !headerSkipped {
			if strings.Contains(line, "Idx") && strings.Contains(line, "Met") {
				headerSkipped = true
			}
			continue
		}
		if line == "" {
			continue
		}
		line = strings.TrimSpace(line)
		fields := regexp.MustCompile(`\s+`).Split(line, -1)
		if len(fields) < 5 {
			continue
		}
		state := fields[3]
		name := fields[4]
		if len(fields) > 5 {
			for i := 5; i < len(fields); i++ {
				name = fmt.Sprintf("%s %s", name, fields[i])
			}
		}

		if state == "connected" {
			val2, err := strconv.Atoi(fields[1])
			if err != nil {
				continue
			}
			network := &NetworkInterface{
				Index:         fields[0],
				met:           uint32(val2),
				status:        state,
				interfaceName: name,
			}
			networkInterfaces = append(networkInterfaces, network)
		}
	}

	if len(networkInterfaces) == 0 {
		return nil, fmt.Errorf("no active network interface found")
	}

	sort.Slice(networkInterfaces, func(i, j int) bool {
		return networkInterfaces[i].met < networkInterfaces[j].met
	})

	return networkInterfaces, nil
}
