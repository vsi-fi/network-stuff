#!/usr/bin/env python3
import re, subprocess,sys
import json,os

MAX_ATTEMPTS = 3

def execute_cmd(cmd):
    res = subprocess.run(cmd.split(), stdout=subprocess.PIPE)
    return(res.stdout.decode('utf-8').split('\n'))

def parse_cfg(file):
    with open(file) as jason:
        config = json.load(jason)
    return(config)

#Naive check to see that the bond has the expected number of interfaces and they match to the config
def check_bond(bond):
    print("Checking bond "+bond)
    member_count = len(config['bonds'][bond]['members'])
    seen_members = [] 
    with open('/proc/net/bonding/'+bond) as status:
        for row in status:
            for bond_member in config['bonds'][bond]['members']:
                check_member_re = re.match('^Slave\sInterface:\s'+bond_member, row)
                if check_member_re:
                    seen_members.append(bond_member)
    if len(seen_members) != member_count:
        print("Bond members on "+bond+" do not match the config.")
        return(1)
    for member in config['bonds'][bond]['members']:
        if member not in seen_members:
            print("Bond "+bond+" is missing expected member " + member)
            return(1)
    return(0)

#Try to create the bond devices. For whatever reason, this can take several attempts
def create_bond(bond):
    if os.path.exists('/proc/net/bonding/'+bond):
        print("Deleting pre-existing bond device " + bond)
        execute_cmd("ip link delete "+bond) 
    execute_cmd("ip link add " + bond + " type bond mode " + config['bonds'][bond]['config'])
    for member in config['bonds'][bond]['members']:
        execute_cmd("ifconfig " + member + " down")        
        execute_cmd("ip link set dev "+member+" master "+bond)
        execute_cmd("ifconfig " + member + " up")        
    execute_cmd("ifconfig " + bond + " mtu "+ config['bonds'][bond]['mtu']+" up")

config = parse_cfg(sys.argv[1])
print("\n\nI'll be attempting to configure network namespaces for you. Note that this tool might spit some warnings as for whatever reason sometimes the bond members go not get assigned correctly on one pass. I will however re-try and check.\n")
for bond in config['bonds']:
    attempts = 0
    create_bond(bond)
    if check_bond(bond) != 0 and attempts <= MAX_ATTEMPTS:
        create_bond(bond)
        attempts += 1
    if attempts == MAX_ATTEMPTS:
        print("Couldn't configure bond " + bond + " after few attempts")

for vlan_device in config['vlans']:
    execute_cmd("ip link add link "+config['vlans'][vlan_device]['parent'] +" name " + vlan_device + " type vlan id "+ config['vlans'][vlan_device]['vlan'])
    execute_cmd("ifconfig "+vlan_device+" mtu " + config['vlans'][vlan_device]['mtu']+" up")

for namespace in config['namespaces']:
    if namespace == "default":
        for interface in config['namespaces'][namespace]['interfaces']:
            execute_cmd("ifconfig "+interface+ " " + config['namespaces'][namespace]['interfaces'][interface]['ifcfg'])
        continue
    execute_cmd('ip netns delete ' + namespace)
    execute_cmd('ip netns add '+namespace)
    for interface in config['namespaces'][namespace]['interfaces']:
        execute_cmd("ip link set "+interface+" netns "+namespace)
        execute_cmd("ip netns exec "+namespace+" ifconfig " + interface + " " + config['namespaces'][namespace]['interfaces'][interface]['ifcfg'])
    for route in config['namespaces'][namespace]['routes']:
        execute_cmd("ip netns exec "+namespace+" route add " + route + " gw " + config['namespaces'][namespace]['routes'][route])
print("\n\nProvisioned network namespaces with interfaces :")
for r in execute_cmd("ip netns"):
    print(r)
    ns_re = re.match('^(.*?)\s', r)
    if ns_re:
        for r2 in execute_cmd('ip netns exec '+ns_re.group(1)+' ifconfig'):
            print(r2)
        print("Namespace has the following routing configured:")
        for route in execute_cmd("ip netns exec "+ns_re.group(1)+" ip route"):
            print(route)
print("You can try to start your application inside the namespace like this: ip netns exec $name_of_the_namespace $command")
