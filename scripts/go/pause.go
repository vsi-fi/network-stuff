package main
//thingy to send out ethernet pauses - used to debug junos evo MAC pause counters in conjunction with pfc sender 2025 / <vsimola@cern.ch>
import (
    "flag"
    "fmt"
    "log"
    "net"
    "os"
    "syscall"
)

//this bit was not part of the library available at the time
func htons(i uint16) uint16 {
            return (i<<8)&0xff00 | i>>8
    }


func main() {
    ifaceName := flag.String("iface", "", "Network interface to send PAUSE frame from")
    flag.Parse()

    if *ifaceName == "" {
        log.Fatalf("Usage: %s -iface <interface_name>", os.Args[0])
    }

    iface, err := net.InterfaceByName(*ifaceName)
    if err != nil {
        log.Fatalf("Failed to get interface %s: %v", *ifaceName, err)
    }

    // Destination MAC for PAUSE frames is 01:80:C2:00:00:01
    dstMAC := []byte{0x01, 0x80, 0xC2, 0x00, 0x00, 0x01}
    srcMAC := iface.HardwareAddr

    // EtherType for Ethernet flow control (PAUSE frame)
    etherType := []byte{0x88, 0x08}

    // Control opcode for PAUSE frame = 0x0001
    opcode := []byte{0x00, 0x01}

    // Pause time (in units of 512 bit times), e.g., 0xFFFF = maximum pause
    pauseTime := []byte{0xFF, 0xFF}

    // Padding to reach minimum Ethernet frame size (64 bytes incl. CRC)
    padding := make([]byte, 42)

    // Construct Ethernet frame
    frame := append(dstMAC, srcMAC...)
    frame = append(frame, etherType...)
    frame = append(frame, opcode...)
    frame = append(frame, pauseTime...)
    frame = append(frame, padding...)

    // Open raw socket
    fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(0x0003)))
    //fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(syscall.HTONS(0x0003)))
    if err != nil {
        log.Fatalf("Failed to open raw socket: %v", err)
    }
    defer syscall.Close(fd)

    // Get interface index
    ifaceIndex := iface.Index

    // Create sockaddr_ll
    sll := &syscall.SockaddrLinklayer{
        Ifindex:  ifaceIndex,
        Halen:    6,
        Addr:     [8]uint8{dstMAC[0], dstMAC[1], dstMAC[2], dstMAC[3], dstMAC[4], dstMAC[5]},
        Protocol: syscall.ETH_P_ALL,
    }

    // Send frame
    err = syscall.Sendto(fd, frame, 0, sll)
    if err != nil {
        log.Fatalf("Failed to send frame: %v", err)
    }

    fmt.Println("PAUSE frame sent.")
}
