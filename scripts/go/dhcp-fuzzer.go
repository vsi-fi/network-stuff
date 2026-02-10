package main
/*
Tool to fuzz dhcp relays/server
vsimola@cern.ch with little help from perplexity.ai <2026>
Usage: $0 -iface vlan114 -vlan 0 -debug, this would send a vlan tagged frame out using the vlan 114. So the vlan tagging bit is little broken at the moment 

To build:
(Make sure you have pcap-devel package of some form installed, pcap.h etc.)
mkdir dhcp-fuzzer
cp dhcp-fuzzer.go dhcp-fuzzer/.
go mod init dhcp-fuzzer
go get github.com/google/gopacket
go get github.com/google/gopacket/layers
go get github.com/google/gopacket/pcap
go get github.com/insomniacslk/dhcp/dhcpv4
go build dhcp-fuzzer.go

*/

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

type customOption struct {
	Code  dhcpv4.OptionCode
	Value []byte
}

type multiStringFlag []string

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func parseOptionCodes(csv string) ([]dhcpv4.OptionCode, error) {
	if strings.TrimSpace(csv) == "" {
		return nil, nil
	}
	parts := strings.Split(csv, ",")
	var res []dhcpv4.OptionCode
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		code, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid option code %q: %w", p, err)
		}
		if code < 0 || code > 255 {
			return nil, fmt.Errorf("option code out of range: %d", code)
		}
		res = append(res, dhcpv4.GenericOptionCode(code))
	}
	return res, nil
}

func parseCustomOptions(args []string) ([]customOption, error) {
	var opts []customOption
	for _, a := range args {
		parts := strings.SplitN(a, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid opt %q, want code:hex", a)
		}
		codeInt, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid opt code in %q: %w", a, err)
		}
		if codeInt < 0 || codeInt > 255 {
			return nil, fmt.Errorf("opt code out of range: %d", codeInt)
		}
		val, err := hex.DecodeString(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid hex payload in %q: %w", a, err)
		}
		opts = append(opts, customOption{
			Code:  dhcpv4.GenericOptionCode(codeInt),
			Value: val,
		})
	}
	return opts, nil
}

func parseClientID(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if strings.ContainsAny(s, ": -") {
		clean := strings.NewReplacer(":", "", " ", "", "-", "").Replace(s)
		b, err := hex.DecodeString(clean)
		if err != nil {
			return nil, fmt.Errorf("clientid hex parse error: %w", err)
		}
		return b, nil
	}
	if len(s)%2 == 0 {
		if _, err := hex.DecodeString(s); err == nil {
			b, _ := hex.DecodeString(s)
			return b, nil
		}
	}
	return []byte(s), nil
}

func randomXID() dhcpv4.TransactionID {
	var xid dhcpv4.TransactionID
	rand.Read(xid[:])
	return xid
}

func buildDHCPPacket(
	iface *net.Interface,
	requestOptCodes []dhcpv4.OptionCode,
	customOpts []customOption,
	hostname string,
	clientID []byte,
	vendorClass string,
	userClassCSV string,
	hops uint8,
	ciaddrStr string,
	giaddrStr string,
	msgType dhcpv4.MessageType,
	randomClientAddr bool,
	debug bool,
) (*dhcpv4.DHCPv4, error) {

	pkt, err := dhcpv4.New()
	if err != nil {
		return nil, err
	}
	pkt.OpCode = dhcpv4.OpcodeBootRequest
	pkt.HWType = 1 // Ethernet
	pkt.HopCount = hops
	pkt.TransactionID = randomXID()
	pkt.Flags = 0x8000 // broadcast bit

	if iface != nil {
		hwAddr := iface.HardwareAddr
		if randomClientAddr == true {
			hwAddr = RandomMAC()
		}
		copy(pkt.ClientHWAddr[:], hwAddr)
	}

	if ciaddrStr != "" {
		pkt.ClientIPAddr = net.ParseIP(ciaddrStr).To4()
	}
	if giaddrStr != "" {
		pkt.GatewayIPAddr = net.ParseIP(giaddrStr).To4()
	}

	// Message type
	pkt.Options.Update(dhcpv4.OptMessageType(msgType))

	// Parameter request list
	if len(requestOptCodes) > 0 {
		pkt.Options.Update(dhcpv4.OptParameterRequestList(requestOptCodes...))
	}

	// Client identifying options
	if hostname != "" {
		pkt.Options.Update(dhcpv4.OptHostName(hostname))
	}
	if len(clientID) > 0 {
		pkt.Options.Update(dhcpv4.OptClientIdentifier(clientID))
	}
	if vendorClass != "" {
		pkt.Options.Update(dhcpv4.OptClassIdentifier(vendorClass))
	}
	if userClassCSV != "" {
		classes := strings.Split(userClassCSV, ",")
		for i := range classes {
			classes[i] = strings.TrimSpace(classes[i])
		}
		for _, class := range classes {
			if class != "" {
				pkt.Options.Update(dhcpv4.OptUserClass(class))
			}
		}
	}

	// Custom options
	for _, co := range customOpts {
		pkt.Options.Update(dhcpv4.OptGeneric(co.Code, co.Value))
	}

	if debug {
		log.Printf("Built %s packet:\n%s", msgType, pkt.Summary())
	}

	return pkt, nil
}

func RandomMAC() net.HardwareAddr {
    buf := make([]byte, 6)
    // Fill with cryptographically secure random bytes
    rand.Read(buf)
    // Set the "locally administered" bit (bit 1 of first byte)
    // This marks it as randomly generated, not globally unique
    buf[0] = buf[0] | 0x02
    // Clear the "group" bit (bit 0 of first byte) to make it unicast
    buf[0] = buf[0] &^ 0x01
    return net.HardwareAddr(buf)
}

func sendPacket(ifaceName string, dhcpPkt *dhcpv4.DHCPv4, targetIPStr string, vlanID uint16, debug bool, randomSrcMac bool) error {
	// Open pcap handle on PARENT interface (no VLAN subinterface!)
	handle, err := pcap.OpenLive(ifaceName, 1600, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("pcap.OpenLive(%s): %v", ifaceName, err)
	}
	defer handle.Close()

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("net.InterfaceByName(%s): %v", ifaceName, err)
	}

	dhcpRaw := dhcpPkt.ToBytes()
	// Build Ethernet layer FIRST
	eth := &layers.Ethernet{
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // broadcast
	}
	if randomSrcMac == true {
		eth.SrcMAC = RandomMAC()
	} else {
		eth.SrcMAC = iface.HardwareAddr
	}

	// IP layer
	ip := &layers.IPv4{
		Version:  4,
		TTL:      128,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    net.IPv4(0, 0, 0, 0),
		DstIP:    net.ParseIP(targetIPStr).To4(),
	}

	udp := &layers.UDP{
		SrcPort: layers.UDPPort(68),
		DstPort: layers.UDPPort(67),
	}

	// CRITICAL: Set IP layer for UDP checksum calculation
	udp.SetNetworkLayerForChecksum(ip)

	// Build layers in CORRECT order: Ethernet -> [VLAN] -> IP -> UDP -> Payload
	var layersToSerialize []gopacket.SerializableLayer
	layersToSerialize = append(layersToSerialize, eth) // Ethernet FIRST
	
	// VLAN layer (if specified) - AFTER Ethernet, BEFORE IP
	if vlanID > 0 && vlanID <= 4095 {
		eth.EthernetType = layers.EthernetTypeDot1Q // VLAN packets
		vlan := &layers.Dot1Q{
			VLANIdentifier: vlanID,
			Priority:       0,
			Type:           layers.EthernetTypeIPv4, // Next protocol
		}
		layersToSerialize = append(layersToSerialize, vlan)
	} else {
		eth.EthernetType = layers.EthernetTypeIPv4 // Untagged packets
	}

	// Network layers AFTER VLAN (or directly after Ethernet for untagged)
	layersToSerialize = append(layersToSerialize, ip, udp, gopacket.Payload(dhcpRaw))

	// Serialize packet
	buffer := gopacket.NewSerializeBuffer()
	options := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err = gopacket.SerializeLayers(buffer, options, layersToSerialize...)
	if err != nil {
		return fmt.Errorf("serialize layers: %v", err)
	}

	// Send packet
	err = handle.WritePacketData(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("write packet: %v", err)
	}

	if debug {
		layerType := "L2"
		if vlanID > 0 {
			layerType = fmt.Sprintf("L2+VLAN%d", vlanID)
		}
		log.Printf("Sent %s packet (XID=0x%08x, %d bytes) on %s", 
			layerType, dhcpPkt.TransactionID, len(buffer.Bytes()), ifaceName)
	}

	return nil
}

func main() {
	var (
		ifaceName      string
		targetIPStr    string
		vlanID         uint
		reqListCSV     string
		hostname       string
		clientIDStr    string
		vendorClass    string
		userClassCSV   string
		hops           uint
		ciaddrStr      string
		giaddrStr      string
		count          int
		interval       time.Duration
		debug          bool
		randomSrcMac   bool
		randomClientAddr   bool
		msgTypeStr     string
		customOptFlags multiStringFlag
	)

	flag.StringVar(&ifaceName, "iface", "", "Interface to use (required)")
	flag.StringVar(&targetIPStr, "target", "255.255.255.255", "Target IP (default broadcast)")
	flag.UintVar(&vlanID, "vlan", 0, "VLAN ID (0=untagged)")
	flag.StringVar(&reqListCSV, "req", "1,3,6,15,51,53,64,252,81,12", "Requested options")
	flag.StringVar(&hostname, "hostname", "", "Hostname (Option 12)")
	flag.StringVar(&clientIDStr, "clientid", "", "Client Identifier")
	flag.StringVar(&vendorClass, "vendorclass", "", "Vendor Class Identifier (Option 60)")
	flag.StringVar(&userClassCSV, "userclass", "", "User Class (Option 77)")
	flag.UintVar(&hops, "hops", 0, "Hop count")
	flag.StringVar(&ciaddrStr, "ciaddr", "", "Client IP address field")
	flag.StringVar(&giaddrStr, "giaddr", "", "Gateway IP address field")
	flag.IntVar(&count, "count", 1, "Number of packets to send")
	flag.DurationVar(&interval, "interval", 0, "Interval between packets. For example: 1s or 10ms etc.")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.BoolVar(&randomSrcMac, "randomsrcmac", false, "Generate random src mac for the dhcp packet.")
	flag.BoolVar(&randomClientAddr, "randomclientmac", false, "Generate random client mac for the dhcp packet payload.")
	flag.StringVar(&msgTypeStr, "msgtype", "Discover", "Message type: Discover, Request, etc.")
	flag.Var(&customOptFlags, "opt", "Custom option code:hexpayload (repeatable)")
	flag.Parse()

	if ifaceName == "" {
		flag.Usage()
		os.Exit(1)
	}

	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Fatalf("failed to get interface %s: %v", ifaceName, err)
	}

	reqCodes, err := parseOptionCodes(reqListCSV)
	if err != nil {
		log.Fatalf("failed to parse -req: %v", err)
	}

	customOpts, err := parseCustomOptions(customOptFlags)
	if err != nil {
		log.Fatalf("failed to parse -opt: %v", err)
	}

	clientIDBytes, err := parseClientID(clientIDStr)
	if err != nil {
		log.Fatalf("failed to parse -clientid: %v", err)
	}

	msgTypeMap := map[string]dhcpv4.MessageType{
		"Discover": dhcpv4.MessageTypeDiscover,
		"Offer":    dhcpv4.MessageTypeOffer,
		"Request":  dhcpv4.MessageTypeRequest,
		"Decline":  dhcpv4.MessageTypeDecline,
		"Ack":      dhcpv4.MessageTypeAck,
		"Nak":      dhcpv4.MessageTypeNak,
		"Release":  dhcpv4.MessageTypeRelease,
		"Inform":   dhcpv4.MessageTypeInform,
	}
	msgType, ok := msgTypeMap[msgTypeStr]
	if !ok {
		log.Fatalf("invalid -msgtype %q", msgTypeStr)
	}

	log.Printf("Sending %d %s packets to %s (VLAN %d) via %s...", count, msgTypeStr, targetIPStr, vlanID, ifaceName)

	for i := 0; i < count; i++ {
		dhcpPkt, err := buildDHCPPacket(iface, reqCodes, customOpts, hostname,
			clientIDBytes, vendorClass, userClassCSV, uint8(hops), ciaddrStr, giaddrStr, msgType, randomClientAddr, debug)
		if err != nil {
			log.Fatalf("failed to build packet: %v", err)
		}

		err = sendPacket(ifaceName, dhcpPkt, targetIPStr, uint16(vlanID), debug, randomSrcMac)
		if err != nil {
			log.Printf("failed to send packet %d: %v", i+1, err)
			continue
		}

		if interval > 0 && i < count-1 {
			time.Sleep(interval)
		}
	}

	log.Println("All packets sent!")
}

