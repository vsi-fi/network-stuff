#!/usr/bin/python3
# -device
#  - vlan_name
#    - list_of_member_ports

import sys, subprocess, re
import json


verbose = 1
snmpwalk = "/usr/bin/snmpwalk"

#devices to test against: in,a,list,of,devices
if len(sys.argv) > 1:
    switches = sys.argv[1].split(',')
else:
    print("Give me a list,of,devices")
    sys.exit(1)

def vprint(msg):
    if(verbose > 0):
        print(msg)

def run_a_thing(cmd):
    return(subprocess.run(cmd.split(" "), stdout=subprocess.PIPE).stdout.decode('utf-8').split('\n'))

def get_interfaces(dev):
    devindex = {}
    for r in run_a_thing(snmpwalk + " -c public -v2c " + dev + " ifDescr"):
        getit = re.match("^.*?ifDescr\.(\d+)\s.*ING:\s(.*?)$", r)
        if getit:
            devindex[getit.group(1)] = getit.groups(2)
    return(devindex)

def get_vlans(dev):
    vlans = {}
    for r in run_a_thing(snmpwalk + " -c public -v2c " + dev + " .1.3.6.1.2.1.17.7.1.4.3.1.1"):
        vlan = re.match(r'^.*?3\.1\.1\.(\d+)\s+.*?ING:\s\"(.*?)\+\1', r)        
        if vlan:
            vlans[vlan.group(1)] = vlan.group(2)
    return(vlans)

def get_vlan_assignments(dev):
    port_vlans = {}
    for r in run_a_thing(snmpwalk + " -c public -v2c " + dev + " .1.3.6.1.2.1.17.7.1.4.3.1.2"):
        m = re.match('^.*?\.2\.(\d+)\s=.*?NG:\s\"(\d+.*?)\"$', r)
        if m:
            port_vlans[m.group(1)] = m.group(2).split(',')
    return(port_vlans)

def match_vlan_ports_to_ports(dev):
    map = {}
    for r in run_a_thing(snmpwalk + " -c public -v2c " + dev + " .1.3.6.1.2.1.17.1.4.1.2"):
        m = re.match('^.*?4\.1\.2\.(\d+).*?R\:\s(\d+)$', r)
        if m:
            map[m.group(1)] = m.group(2)
    return(map)

def pp(stuff):
    print(json.dumps(stuff, indent = 4))

blob = dict()
for s in switches:
    blob[s] = {}
    interfaces = get_interfaces(s)
    vlan_port_to_port = match_vlan_ports_to_ports(s)
    vlans = get_vlans(s)
    vlans_on_ports = get_vlan_assignments(s)    

    blob[s]['vlans'] = {}
    for id,name in vlans.items():
        if id in vlans_on_ports:
            blob[s]['vlans'][name] = []
            for p in vlans_on_ports[id]:
                if p in vlan_port_to_port:
                    blob[s]['vlans'][name].append(interfaces[vlan_port_to_port[p]][1])


pp(blob)


