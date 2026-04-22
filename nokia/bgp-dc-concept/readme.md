# Advertising routes with ebgp only
Quick test to see how routes could be advertised to and from data centre with ebgp only.
## Goals
"Upstream" is to send 0/0 route with statically configured session with a 16 bit ASN. Sessions "inside" the DC are based on dynamic neighbors and use 32 bit ASN.
We want to send only aggregate route towards upstream and the AS_PATH path for the aggregate should only contain the 16 bit AS assigned by upstream.
## Configuration
List of relevant configurations, full configs in full_configs
### Upstream router
```
info network-instance default protocols bgp
    autonomous-system 64512
    router-id 1.1.255.255
    afi-safi ipv4-unicast {
        admin-state enable
    }
    group DC1 {
        peer-as 64513
        export-policy [
            IPV4-EXPORT-DC1 // Policy that sends only default
        ]
        import-policy [
            IPV4-IMPORT-DC1 // Policy that accepts the DC1 prefix and discards the rest
        ]
    }
    neighbor 10.255.255.1 {
        admin-state enable
        peer-group DC1
    }
    neighbor 10.255.255.3 {
        admin-state enable
        peer-group DC1
    }
```
### Aggr1 - facing the upstream and the rest of the DC
```
// specify the DC aggregate route. Not sure if summary-only is required as we're anyway going to apply export policies?
info network-instance default aggregate-routes
    route 192.168.0.0/16 {
        admin-state enable
        summary-only true
    }
info network-instance default protocols bgp
    admin-state enable
    autonomous-system 64513 // ASN assigned by upstream
    router-id 1.1.1.1
    dynamic-neighbors {
        interface ethernet-1/1.100 {
            peer-group DC-LEAF
            allowed-peer-as [
                4201000002..4201000099 // accept sessions from range of access ASNs
            ]
        }
        interface ethernet-1/2.100 {
            peer-group DC-LEAF
            allowed-peer-as [
                65502 // or just an individual ASN
            ]
        }
    }
    afi-safi ipv4-unicast {
        admin-state enable
        ipv4-unicast {
            advertise-ipv6-next-hops true // Advertise ipv4 routes with ipv6 next-hop
            receive-ipv6-next-hops true // Same, but opposite, RFC 8950 / RFC 5549
        }
    }
    group DC-LEAF {
        export-policy [
            IPV4-BGP-EXPORT // This policy sends out only 0/0 route and rejects everything else
        ]
        import-policy [
            IPV4-BGP-IMPORT-DC1 // This policy accepts routes fitting into the DC aggregate prefix 192.168.0.0/16
        ]
        afi-safi ipv4-unicast {
        }
        local-as {
            as-number 4201000001 // We're using this AS to talk to the access devices in the DC
        }
        transport {
            passive-mode true // We're not actively attempting to form sessions, instead it is up to the access devices to try this
        }
    }
    group UPSTREAM {
        peer-as 64512
        export-policy [
            IPV4-BGP-EXPORT-UPSTREAM // This policy sends the aggregate route 192.168.0.0/16 upstream and rejects everything else
        ]
        import-policy [
            IPV4-BGP-IMPORT-UPSTREAM // This policy accepts the 0/0 route and rejects the rest
        ]
        local-as {
            as-number 64513 // Again, the AS assigned by upstream - this could likely be delete from here
        }
    }
    neighbor 10.255.255.0 {
        peer-group UPSTREAM // Finally, specify the upstream neighbor and assign it to a peer group
    }
info network-instance default ip-forwarding
     receive-ipv4-check false // this is required. Otherwise routes find their way into routing table they cannot be used for forwarding
--{ candidate shared default }--[  ]--

```
### Access router - device where hosts etc. would connect
This is largely just a repeat of the above
```
info network-instance default protocols bgp
    admin-state enable
    autonomous-system 4201000099
    router-id 1.1.1.3
    dynamic-neighbors {
        interface ethernet-1/1.100 {
            peer-group DC-AGGR
        }
        interface ethernet-1/2.100 {
            peer-group DC-AGGR
        }
    }
    afi-safi ipv4-unicast {
        admin-state enable
        ipv4-unicast {
            advertise-ipv6-next-hops true
            receive-ipv6-next-hops true
        }
    }
    group DC-AGGR {
        peer-as 4201000001 // AS of the aggregation device(s)
        export-policy [
            IPV4-BGP-EXPORT // Policy that sends the local networks(host facing, loopbacks etc) of interest and reject the rest
        ]
        import-policy [
            IPV4-BGP-IMPORT // We accept only 0/0 rejecting rest
        ]
        afi-safi ipv4-unicast {
        }
        local-as {
            as-number 4201000099
        }
    }
info network-instance default ip-forwarding
     receive-ipv4-check false // this is required. Otherwise routes find their way into routing table they cannot be used for forwarding
--{ candidate shared default }--[  ]--

```
### Verification
Upstream sees only the aggr1 64513 in the AS_PATH
```
A:admin@upstream# show network-instance default protocols bgp neighbor 10.255.255.1 received-routes ipv4
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Peer        : 10.255.255.1, remote AS: 64513, local AS: 64512
Type        : static
Description : None
Group       : DC1
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Status codes: u=used, *=valid, >=best, x=stale, b=backup, w=unused-weight-only
Origin codes: i=IGP, e=EGP, ?=incomplete
+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
|       Status                Network               Path-id              Next Hop                 MED                 LocPref               AsPath                Origin        |
+===============================================================================================================================================================================+
|         u*>           192.168.0.0/16        0                     10.255.255.1                   -                                  [64513]                        i          |
+-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------+
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
1 received BGP routes : 1 used 1 valid
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------

--{ candidate shared default }--[  ]--
```
We have readability from access device to the loopback of the upstream
```
--{ candidate shared default }--[  ]--
A:admin@upstream# show interface lo0
========================================================================================================================================================================================
lo0 is up, speed None, type None
  lo0.100 is up
    Network-instances:
      * Name: default (default)
    Encapsulation   : null
    Type            : None
    IPv4 addr    : 10.1.255.255/32 (static, preferred, primary)
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
========================================================================================================================================================================================

--{ candidate shared default }--[  ]--
A:admin@leaf-1# show network-instance default ipv4 route 10.1.255.255
========================================================================================================================================================================================
IPv4-unicast route table for default network-instance
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Flags: > (best), * (unviable), ! (failed)
     : L (leaked route from another network-instance)
     : B (backup NHG active and displayed)
     : S (statistics supported)
     : D (dynamic LB), R (resilient LB)
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
Prefix               Route Type   Metric   Pref    Flags    Next-Hop(s)
----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------
0.0.0.0/0            bgp          0        170     >        fe80::181b:ff:feff:1(ethernet-1/1.100)

--{ candidate shared default }--[  ]--

A:admin@leaf-1# info interface lo0
    subinterface 100 {
        ipv4 {
            admin-state enable
            address 192.168.100.1/32 {
            }
        }
    }


A:admin@leaf-1# ping network-instance default 192.168.255.1 -I 192.168.100.1
Using network instance default
PING 192.168.255.1 (192.168.255.1) from 192.168.100.1 : 56(84) bytes of data.
64 bytes from 192.168.255.1: icmp_seq=1 ttl=64 time=3.51 ms
64 bytes from 192.168.255.1: icmp_seq=2 ttl=64 time=2.73 ms
^C
--- 192.168.255.1 ping statistics ---
2 packets transmitted, 2 received, 0% packet loss, time 1001ms
rtt min/avg/max/mdev = 2.732/3.122/3.512/0.390 ms

--{ candidate shared default }--[  ]--
```
