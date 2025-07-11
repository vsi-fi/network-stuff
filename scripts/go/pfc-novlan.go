package main
/* Send pfc pauses, vlans to be tested/looked into whilst avoiding having to congest a network
Use in conjunction with tcpdump and tcprelay if having to deal with TAC
<vsimola@hc.nrec>
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
)

func main() {
	//ifaceName := "data0"
	device := flag.String("device", "VLAN114", "Network interface that we'll try to use to send the pauses.")
	prio_to_pause := flag.String("prio", "all", "Which priorities to pause [0-7], default to all")
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

	// Ethernet layer with EtherType for MAC Control
	ether := &layers.Ethernet{
		SrcMAC:	srcMAC,
		DstMAC:	dstMAC,
		EthernetType: layers.EthernetType(0x8808), // MAC Control, https://en.wikipedia.org/wiki/Ethernet_flow_control
	}

	//PFC pause payload
	payload := make([]byte, 2+2+16) // opcode + enable vector + pause times
	binary.BigEndian.PutUint16(payload[0:2], 0x0101) // PFC opcode 101 means 'you better back of or else
	payload[2] = 0xFF 

	if *prio_to_pause == "all" {
		for i := 0; i < 8; i++ {
			binary.BigEndian.PutUint16(payload[3+i*2:], 0xFFFF) // Max pause time per priority
		}
	} else {
	p, err := strconv.Atoi(*prio_to_pause)
	if err != nil || p < 0 || p > 7 {
		log.Fatalf("Invalid priority: %v", *prio_to_pause)
	}

	binary.BigEndian.PutUint16(payload[2:4], 1 << uint(p)) // Enable only selected priority

	offset := 4 + p*2
	binary.BigEndian.PutUint16(payload[offset:offset+2], 0xFFFF)

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

	// Send the packet
	err = handle.WritePacketData(buffer.Bytes())
	if err != nil {
		log.Fatal("Send failed:", err)
	}

	log.Println("PFC pause frame sent successfully (no VLAN) using device ", *device, " with priority ", *prio_to_pause)
}

func getMAC(ifaceName string) net.HardwareAddr {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Fatal(err)
	}
	return iface.HardwareAddr
}
