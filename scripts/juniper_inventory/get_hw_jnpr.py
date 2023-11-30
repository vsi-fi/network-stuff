#!/usr/bin/python3
#Produce a json document of hw pieces installed in a switch
# - device
#  - item
#    - serial
#     - part_number
#     - description

import sys, subprocess, re
import json

verbose = 1
snmpwalk = "/usr/bin/snmpwalk"

if len(sys.argv) > 1:
    switches = sys.argv[1].split(',')
else:
    print("Give me a,list,of,devices")
    sys.exit(1)

def vprint(msg):
    if(verbose > 0):
        print(msg)

def get_components(dev):
    if dev not in blob:
        blob[dev] = {}

    whatwewant = {'6': 'item', '7': 'serial', '10': 'part_number', '14': 'description'} 
    ids = ['6','7','10','14']
    keys = {}
    for id in ids:
        res = subprocess.run([snmpwalk, "-c", "public", "-v2c", dev, "1.3.6.1.4.1.2636.3.1.8.1."+id], stdout=subprocess.PIPE)
        vals = dict()
        for r in res.stdout.decode('utf-8').split('\n'):
            pickup = re.match('^.*?3.1.8.1.\d+\.(.*?)\s+.*?STRING:\s\"(.*?)\"', r)
            if pickup and id == '6':
                blob[dev][pickup.group(2)] = {}
                keys[pickup.group(1)] = pickup.group(2)
            if pickup and id != '6':
                if pickup.group(1) in keys:
                    blob[dev][keys[pickup.group(1)]][whatwewant[id]] = pickup.group(2)
                

def pp(stuff):
    print(json.dumps(stuff, indent = 4))

blob = dict()
for s in switches:
    get_components(s)
pp(blob)


