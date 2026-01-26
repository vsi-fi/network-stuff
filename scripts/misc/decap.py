#!/usr/bin/env python3
import argparse
from scapy.all import rdpcap, wrpcap, Ether, GRE

def process_pcap(infile, outfile, verbose, shim_len=48, erspan_header_len=8):
    pkts_in  = rdpcap(infile)
    pkts_out = []

    for cp in pkts_in:
        #remove the shim
        raw      = bytes(cp)
        if len(raw) <= shim_len:
            continue  #too short, skip
        raw_no_shim = raw[shim_len:]

        #parse again as an Ethernet frame
        p = Ether(raw_no_shim)

        #decap the de-shimmed packet
        if GRE in p:
            erspan_and_inner = p[GRE].payload          # ERSPAN + inner Ether
            raw_inner = bytes(erspan_and_inner)[erspan_header_len:]
            inner_eth = Ether(raw_inner)
            if verbose is True:
                print(inner_eth)
            pkts_out.append(inner_eth)

    wrpcap(outfile, pkts_out)

def main():
    parser = argparse.ArgumentParser(description="Decapsulate ERSPAN/L2oGRE and write inner Ethernet frames.")
    parser.add_argument("-i", "--input", required=True,help="Input pcap file with ERSPAN/L2oGRE traffic")
    parser.add_argument("-o", "--output", required=True, help="Output pcap file with decapsulated Ethernet frames")
    parser.add_argument("--erspan-len",type=int, default=8,help="ERSPAN Type II header length in bytes (default: 8)")
    parser.add_argument("--shim-len", type=int,default=48, help="Shim header length of the input file" )
    parser.add_argument("--verbose", type=bool,default=False, help="Print stuff" )
    args = parser.parse_args()
    process_pcap(args.input, args.output, args.verbose, args.shim_len, args.erspan_len)

if __name__ == "__main__":
    main()

