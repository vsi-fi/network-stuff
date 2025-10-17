#!/usr/bin/python3
import yaml
import argparse

parser = argparse.ArgumentParser(description="generate set (info flat) style configurations for P1 ebgp")
parser.add_argument("--file", required=True, type=str, help="Name of the yaml file i should try to parse")
args = parser.parse_args()

with open(args.file, "r") as file:
    config = yaml.safe_load(file)

for element,data in config.items():
    if element == "system_configuration":
        for attribute, value in data.items():
            print("set / system configuration " + attribute + " " + value)

    if element == "interfaces":
        for interface in data:
            for attribute,value in interface.items():
                if attribute == "name":
                    continue 
                if attribute == "subinterfaces":
                    for unit in value:
                        print("set / interface "+interface['name']+" subinterface "+str(unit['unit'])+ " admin-state enable")
                        if 'ipv6' in unit:
                            if unit['ipv6']['router_advertisement'] == True:
                                print("set / interface "+interface['name']+" subinterface "+str(unit['unit'])+ " ipv6 admin-state "+str(unit['ipv6']['admin-state']))
                                print("set / interface "+interface['name']+" subinterface "+str(unit['unit'])+ " ipv6 router-advertisement router-role admin-state enable")
                        if 'vlan' in unit:
                            print("set / interface "+interface['name']+" subinterface "+str(unit['unit'])+ " vlan encap single-tagged vlan-id "+str(unit['vlan']))
                        if 'type' in unit and unit['type'] == 'bridged':
                            print("set / interface "+interface['name']+" subinterface "+str(unit['unit'])+ " type " + unit['type']) 
                        if 'ipv4' in unit or 'type' in unit and unit['type'] != 'bridged':
                            print("set / interface "+interface['name']+" subinterface "+str(unit['unit'])+ " ipv4 admin-state "+str(unit['ipv4']['admin-state']))
                            if "address" in unit['ipv4']:
                                print("set / interface "+interface['name']+" subinterface "+str(unit['unit'])+" ipv4 address " + unit['ipv4']['address'])
                            if "unnumbered" in unit['ipv4']:
                                print("set / interface "+interface['name']+" subinterface "+str(unit['unit'])+" ipv4 unnumbered interface " + unit['ipv4']['unnumbered'] + " admin-state enable")

                if type(value) != dict and attribute != "subinterfaces" and attribute != "mtu":
                    print("set / interface "+interface['name']+" "+attribute+" "+str(value))
    if element == "routing_policies":
        for policy in data:
            for term in policy['terms']:
                print("set / routing-policy policy " +policy['name']+ " statement " + str(term['term'])+" action policy-result " + term['action'])
                for condition in term['match']:
                    for attr,value in condition.items():
                        print("set / routing-policy policy " +policy['name']+ " statement " + str(term['term'])+" match "+attr+ " " + value)
    if element == "vrfs":
        for vrf in data:
            if vrf['type'] == "mac-vrf":
                print("set / network-instance "+vrf['name']+" type "+vrf['type'])
                for interface in vrf['interfaces']:
                    print("set / network-instance "+vrf['name'] + " interface " + interface)

            if vrf['type'] == "ip-vrf":
                print("set / network-instance "+vrf['name']+" protocols bgp autonomous-system "+str(vrf['autonomous-system']))
                print("set / network-instance "+vrf['name']+" protocols bgp admin-state enable")
                print("set / network-instance "+vrf['name']+" protocols bgp afi-safi ipv4-unicast admin-state enable multipath maximum-paths 8")
                print("set / network-instance "+vrf['name']+" protocols bgp router-id "+vrf['router_id'])
                print("set / network-instance "+vrf['name']+" type "+vrf['type'])
                for interface in vrf['interfaces']:
                    print("set / network-instance "+vrf['name'] + " interface " + interface)
                for bgp_group in vrf['bgp']['groups']:
                    if "send_ipv4_default" in bgp_group:
                        print("set / network-instance "+vrf['name']+" protocols bgp group " + bgp_group['name'] + " send-default-route ipv4-unicast true")
                    print("set / network-instance "+vrf['name']+" protocols bgp group " + bgp_group['name'] + " export-policy " + bgp_group['export-policy'])
                    print("set / network-instance "+vrf['name']+" protocols bgp group " + bgp_group['name'] + " import-policy " + bgp_group['import-policy'])
                    print("set / network-instance "+vrf['name']+" protocols bgp group " + bgp_group['name'] + " admin-state " + bgp_group['admin_state'])
                    for afi in bgp_group['afi']:
                        print("set / network-instance "+vrf['name']+" protocols bgp group " + bgp_group['name'] + " afi-safi " + afi + " admin-state enable")
                for group in vrf['bgp']['dynamic-neigbors']['groups']:
                    if 'enable_bfd' in group:
                        print("set / network-instance "+vrf['name']+" protocols bgp group "+group['name']+" failure-detection enable-bfd " +group['enable_bfd'])
                    for interface in group['interfaces']:
                        print("set / network-instance "+vrf['name'] + " protocols bgp dynamic-neighbors interface " + interface + " allowed-peer-as " + group['allowed_as'] + " peer-group " + group['name'])#vrf['bgp']['dynamic-neigbors']['groups'][group]['allowed_as']+" peer-group " + vrf['bgp']['dynamic-neigbors']['group'])
