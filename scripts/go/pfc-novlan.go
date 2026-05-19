package main
/* quick and dirty to send pfc pauses <vesa.simola@cern.ch>
To build:
go mod init pfc
#Install dependencies
Make sure you have libpcap-devel installed via your packet manager of choice
go get github.com/google/gopacket
go get github.com/google/gopacket/layers
go get github.com/google/gopacket/pcap
go build pfc.go
*/
import (
	"encoding/binary"
	"log"
	"net"
	"flag"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"strconv"
	"time"
)

func main() {
	//ifaceName := "data0"
	device := flag.String("device", "VLAN114", "Network interface that we'll try to use to send the pauses. This should be a physical interface")
	prio_to_pause := flag.String("prio", "all", "Which priorities to pause [0-7], default to all")
	num_of_pkts := flag.Int("num_of_pkts",10, "How many packets to send?")
	//interval := flag.Int("interval", 1000, "Sleep in milliseconds between sending packets")
	interval := flag.Int("interval", 5000, "Sleep in *nanooseconds* between sending packets")
	ethertype_str := flag.String("ethertype", "0x8808", "Ethertype to use if trying to mess with the device")
	opcode_str := flag.String("opcode", "0x0101", "Opcode to use")
	quanta_str := flag.String("quanta", "0xFFFF", "How long to pause?")
	dst_mac_str := flag.String("dst_mac", "01:80:C2:00:00:01", "destination mac address to use")
	flag.Parse()
	// Open the network interface for packet injection
	handle, err := pcap.OpenLive(*device, 65536, false, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// Get MAC addresses
	srcMAC := getMAC(*device)
	//https://en.wikipedia.org/wiki/Ethernet_flow_control
	dstMAC := net.HardwareAddr{0x01, 0x80, 0xC2, 0x00, 0x00, 0x01} // PFC multicast MAC
	if *dst_mac_str != "" {
		parsedMAC, err := net.ParseMAC(*dst_mac_str)
		if err != nil {
			log.Fatalf("Invalid MAC address %q: %v", *dst_mac_str, err)
		}
		dstMAC = parsedMAC
	}

	ethertype, err := strconv.ParseUint((*ethertype_str)[2:], 16, 16)
	opcode_, err := strconv.ParseUint((*opcode_str)[2:], 16, 16)
	opcode := uint16(opcode_)

	quanta_, err := strconv.ParseUint((*quanta_str)[2:], 16, 16)
	quanta := uint16(quanta_)

	//ethertype_to_use := layers.EthernetType(ethertype)
	// Ethernet layer with EtherType for MAC Control
	ether := &layers.Ethernet{
		SrcMAC:	srcMAC,
		DstMAC:	dstMAC,
		EthernetType: layers.EthernetType(ethertype), // MAC Control, https://en.wikipedia.org/wiki/Ethernet_flow_control
	}

	//PFC pause payload
	payload := make([]byte, 2+2+16) // opcode + enable vector + pause times
	binary.BigEndian.PutUint16(payload[0:2], opcode) // PFC opcode 101 means pfc 000 for flowcontrol
	payload[2] = 0xFF

	if *prio_to_pause == "all" {
		for i := 0; i < 8; i++ {
			binary.BigEndian.PutUint16(payload[3+i*2:], quanta)
		}
	} else {
	p, err := strconv.Atoi(*prio_to_pause)
	if err != nil || p < 0 || p > 7 {
		log.Fatalf("Invalid priority: %v", *prio_to_pause)
	}

	binary.BigEndian.PutUint16(payload[2:4], 1 << uint(p)) // Enable only selected priority

	offset := 4 + p*2
	binary.BigEndian.PutUint16(payload[offset:offset+2], quanta)

	}

	// Serialize a packet
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}

	err = gopacket.SerializeLayers(buffer, opts,
		 ether,
		gopacket.Payload(payload),
	)
	if err != nil {
		log.Fatal("Serialization failed:", err)
	}

	// Send packets
	count := 0
	status := *num_of_pkts / 10
	for count < *num_of_pkts {
		if count > 10 &&  count%status == 0 {
			log.Println("Sending packet number", count)
		} else if count <= 10 {
			log.Println("Sending packet number", count)
		}
		err = handle.WritePacketData(buffer.Bytes())
		if err != nil {
			log.Fatal("Send failed:", err)
		}
		//time.Sleep(time.Duration(*interval) * time.Millisecond)
		time.Sleep(time.Duration(*interval) * time.Nanosecond)
		count++
	}

	log.Println("PFC pause frame(s) sent successfully using device ", *device, " with priority ", *prio_to_pause)
}

func getMAC(ifaceName string) net.HardwareAddr {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Fatal(err)
	}
	return iface.HardwareAddr
}
