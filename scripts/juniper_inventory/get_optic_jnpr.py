#!/usr/bin/python3

#Produce json document of optics installed in a Junos box
#- port
#- physical_info
#  - type
#   - serial_number
#   - manufacturer
#   - phys_dev_snmp_id
# - optical_info
#   - current_rx_power
#   - temperature
#   - alarm_thresholds

import sys, subprocess, re
import json

verbose = 1
snmpwalk = "/usr/bin/snmpwalk"

#devices to test against: in,a,list,of,devices
if len(sys.argv) > 1:
    switches = sys.argv[1].split(',')
else:
    print("Give me a,list,of,devices")
    sys.exit(1)

def vprint(msg):
    if(verbose > 0):
        print(msg)

def execute_snmp_walk(query, dev):
    res = subprocess.run([snmpwalk, "-c", "public", "-v2c", dev, query], stdout=subprocess.PIPE)
    vals = dict()
    if query == "1.3.6.1.2.1.31.1.1.1.1":
        for r in res.stdout.decode('utf-8').split('\n'):
            data = re.match('^.*?ifName\.(\d+)\s+.*?ING:\s(.*?)$', r)
            try:
                vals[data.group(1)] = data.group(2)
            except:
                pass
        return(vals)
    if query == "1.3.6.1.4.1.2636.3.60.1.1.1":
        for r in res.stdout.decode('utf-8').split('\n'):
            data = re.match('^.*?\.1\.1\.1\.1\.(\d+)\.(\d+)\s=\s.*?(?:ING|GER)\:\s(.*?)$', r)
            try:
                (metric, index, value) = data.group(1), data.group(2), data.group(3)

                #https://apps.juniper.net/mib-explorer/navigate?software=Junos%20OS&release=22.3R3&name=jnxDomCurrentRxLaserPower&oid=1.3.6.1.4.1.2636.3.60.1.1.1.1

                if index not in vals:
                    vals[index] = dict()

                if metric == "5":
                    vals[index]['jnxDomCurrentRxLaserPower'] = value 

                if metric == "10":
                    vals[index]['jnxDomCurrentRxLaserPowerLowAlarmThreshold'] = value

                if metric == "12":
                    vals[index]['jnxDomCurrentRxLaserPowerLowWarningThreshold'] = value

                if metric == "8":
                    vals[index]['jnxDomCurrentModuleTemperature'] = value

                if metric == "23":
                    vals[index]['jnxDomCurrentModuleTemperatureHighWarningThreshold'] = value

                if metric == "21":
                    vals[index]['jnxDomCurrentModuleTemperatureHighAlarmThreshold'] = value

            except:
                pass
        return(vals)

    if query == "1.3.6.1.2.1.47.1.1.1.1":
        txcvr = dict()
        txcvr_by_location = dict()
        for r in res.stdout.decode('utf-8').split('\n'):
            data = re.match('^.*?\.1\.1\.1\.1\.(\d+)\.(\d+).*?\"(.*?)\"', r)
            try:
                (info,type,value) = data.group(1), data.group(2), data.group(3) 
                if info == "11":
                    txcvr[data.group(2)]['serial_number'] = data.group(3)

                if info == "12":
                    txcvr[data.group(2)]['manufacturer'] = str(data.group(3)).strip()

                if re.match('^.*?\s@\s', value):
                    loc = re.match('^(.*?)\s@\s(.*?)$', value)
                    txcvr_type = loc.group(1)
                    #Junos and Junos EVO seem to differ in the syntax they give for the location
                    #Junos: 0/0/48
                    #Junos EVO: /Chassis[0]/Fpc[0]/Pic[0]/Port[0]
                    #Try to normalise to Junos
                    if re.match('^\/Chassis.*?Port', loc.group(2)):
                            evo_loc = re.match('^.*?Fpc\[(\d+)\]/Pic\[(\d+)\]/Port\[(\d+)\]', loc.group(2))
                            location = evo_loc.group(1) + "/" + evo_loc.group(2) + "/" + evo_loc.group(3)
                    else:
                        location = loc.group(2)
                    if re.match('\d+/\d+/\d+', location):
                        if data.group(2) not in txcvr:
                            txcvr[data.group(2)] = dict()
                        txcvr[data.group(2)]['location'] = location
                        txcvr[data.group(2)]['type'] = txcvr_type 
            except:
                pass

        for s_index, data in txcvr.items():
            txcvr_by_location[txcvr[s_index]['location']] = dict()
            txcvr_by_location[txcvr[s_index]['location']] = data 
            txcvr_by_location[txcvr[s_index]['location']]['phys_dev_snmp_id'] = s_index
            txcvr_by_location[txcvr[s_index]['location']].pop('location')

    return(txcvr_by_location)

def pp(stuff):
    print(json.dumps(stuff, indent = 4))

def get_interface_index(box):
   installed_hw = execute_snmp_walk("1.3.6.1.2.1.47.1.1.1.1", box) 
   ints = execute_snmp_walk("1.3.6.1.2.1.31.1.1.1.1", box)
   metrics = execute_snmp_walk("1.3.6.1.4.1.2636.3.60.1.1.1", box)

   blob[box] = dict()
   for snmp_index, interface in ints.items():
       i_name = re.match('^.*?(\d+\/.*?)(?::|$)', interface)
       if i_name and i_name.group(1) in installed_hw:
           if snmp_index in metrics:
               blob[box][interface] = dict()
               blob[box][interface]['physical_info'] = installed_hw[i_name.group(1)]
               blob[box][interface]['optical_info'] = metrics[snmp_index]
               blob[box][interface]['snmp_index'] = snmp_index
           else:
               blob[box][interface] = dict()
               blob[box][interface]['physical_info'] = installed_hw[i_name.group(1)]
               blob[box][interface]['snmp_index'] = snmp_index

blob = dict()
for s in switches:
    get_interface_index(s)
pp(blob)


