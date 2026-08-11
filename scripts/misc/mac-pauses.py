#!/usr/bin/env python3
#Send ethernet pauses, possibly with funny options
#vesa.simola@cern.ch / 2026
import argparse
import time
from scapy.all import Ether, sendp, Raw

def send_pause_frames(interface, rate_ms, ethertype, opcode, timeslots, dstmac):
    pause_frame = Ether(dst=dstmac, type=int(ethertype, 16)) / Raw(
        load=bytes.fromhex(opcode + " " + timeslots)
    )
    try:
        sent_count = 0
        while True:
            sendp(pause_frame, iface=interface, verbose=False)
            sent_count += 1
            print(str(sent_count) + " pauses sent")
            time.sleep(rate_ms / 1000.0)
    except KeyboardInterrupt:
        print("\nStopping pause frame transmission.")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Send Ethernet pause frames at a specified rate.")
    parser.add_argument("-i", "--interface", required=True, help="Network interface to send pause frames from")
    parser.add_argument("-r", "--rate", type=int, required=True, help="Rate in milliseconds between pause frames")
    parser.add_argument("-e", "--ethertype", type=str, required=False, default="0x8808", help="Ethertype to be used")
    parser.add_argument("-o", "--opcode", type=str, required=False, default="0001", help="Opcode to be used")
    parser.add_argument("-t", "--timeslots", type=str, required=False, default="ffff ffff ffff ffff ffff ffff ffff", help="Timeslots to be used")
    parser.add_argument("-d", "--dstmac", type=str, required=False, default="01:80:c2:00:00:01", help="Destination mac where to send the pauses")
    args = parser.parse_args()

    print(f"Sending pause frames on {args.interface} every {args.rate}ms. Press Ctrl+C to stop.")
    send_pause_frames(args.interface, args.rate, args.ethertype, args.opcode, args.timeslots, args.dstmac)
