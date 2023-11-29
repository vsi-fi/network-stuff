# eBGP peers with no peer IP addressing statically configured #
Per default a BGP peering needs either the "allow $subnet" -statement or "neighbor $ip" -statement to form a session.

Whilst I've used the "allow $subnet passive" in the past with some success to slim down the BGP configs it would seem that there is a better method to this.

I'll try to describe the dynamic-neighbor -feature and how to apply that to an underlay for EVPN.

## Building blocks ##

This stuff relies on few extra bits to what I've used before:

* IPv6 router advertisements
* peer-auto-discovery -knob in [ protocols bgp group $GROUP_NAME ]

### IPv6 router advertisements, neighbor discovery protocol (NDP) ###

Device send periodic router advertisements (type 134, ICMPv6, multicast) and additionally explicit responses to solicitations (type 133, ICMPv6) from other devices in the same layer two domain.
These exchanges are used to learn the whereabouts of routers in the same segment. 

Bit similarly to ARP on IPv4, the neighbor solicitation (type 135, ICMPv6) can be used to learn the L2 address of a device in the same layer two domain.

Router advertisements have to be explicitly configured on a Junos device:

```
show configuration protocols router-advertisement 
interface ge-0/0/1.0;
interface ge-0/0/2.0;

```

In its simplest form we just turn on the said functionality as above.

In addition to the protocol, one needs to add inet and inet6 as address-families on the interface:

```
show interfaces ge-0/0/1  
unit 0 {
    family inet;
    family inet6;
}

[edit]
```

With the above things configured the neighbor discovery should mostly work:

```
show ipv6 neighbors    
IPv6 Address                            Linklayer Address  State       Exp   Rtr  Secure  Interface               
fe80::e20:51ff:fed2:1                    0c:20:51:d2:00:01  reachable   12    yes  no      ge-0/0/1.0              
fe80::e5e:4dff:fe90:1                    0c:5e:4d:90:00:01  reachable   28    yes  no      ge-0/0/2.0              
Total entries: 2

[edit]

```

There seems to be some knobs to avoid accidental router-advertisements from impacting the network. I'll see if can test those later.

### BGP dynamic-neighbor ###

With the router advertisements going between the boxes we can flip on BGP:

```
show protocols bgp   
group UNDERLAY {
    type external;
    family inet {
        unicast {
            /* Needed to allow advertisements of IPv4 prefixes with IPv6 next-hop */
            extended-nexthop;
        }
    }
    export EXPORT-BGP;
    peer-as 65001;
    local-as 65000;
    /* Needed since the access switches are all using local-as of 65001: per default prefixes are not sent back to the same AS
    Not to mention the AS loop preventing the access switches from importing the route. There are other ways to work around this as well */
    as-override;
    /* Instead of defining the neighbor address, we try to form a session based on router-advertisements */
    dynamic-neighbor bgp_unnumbered {
        peer-auto-discovery {
            family inet6 {
                ipv6-nd;
            }
            /* Try forming session based on adverts seen on these interfaces */
            interface ge-0/0/1.0;
            interface ge-0/0/2.0;
        }
    }
}

[edit]

```
One can verify the BGP status the usual way:

```
root@CORE-1# run show bgp summary 

Warning: License key missing; requires 'bgp' license

Threading mode: BGP I/O
Default eBGP mode: advertise - accept, receive - accept
Groups: 1 Peers: 2 Down peers: 0
Auto-discovered peers: 2
Table          Tot Paths  Act Paths Suppressed    History Damp State    Pending
inet.0               
                       2          2          0          0          0          0
Peer                     AS      InPkt     OutPkt    OutQ   Flaps Last Up/Dwn State|#Active/Received/Accepted/Damped...
fe80::e20:51ff:fed2:1%ge-0/0/1.0       65001         21         20       0       2        7:50 Establ
  inet.0: 1/1/1/0
fe80::e5e:4dff:fe90:1%ge-0/0/2.0       65001         21         20       0       1        7:50 Establ
  inet.0: 1/1/1/0

[edit]

root@CORE-1# run show bgp neighbor fe80::e20:51ff:fed2:1 

Warning: License key missing; requires 'bgp' license

Peer: fe80::e20:51ff:fed2:1%ge-0/0/1.0+50172 AS 65001 Local: fe80::eb7:ff:fe3f:2%ge-0/0/1.0+179 AS 65000
  Group: UNDERLAY              Routing-Instance: master
  Forwarding routing-instance: master  
  Type: External    State: Established    Flags: <Sync AutoDiscoveredNdp>
  Last State: OpenConfirm   Last Event: RecvKeepAlive
  Last Error: Cease
  Export: [ EXPORT-BGP ] 
  Options: <AddressFamily PeerAS LocalAS Refresh As Override>
  Options: <GracefulShutdownRcv>
  Address families configured: inet-unicast
  Holdtime: 90 Preference: 170
  Graceful Shutdown Receiver local-preference: 0
  Local AS: 65000 Local System AS: 0
  Number of flaps: 2
  Last flap event: Stop
  Receive eBGP Origin Validation community: Reject
  Error: 'Cease' Sent: 1 Recv: 1
  Peer ID: 10.1.1.2        Local ID: 10.1.1.1          Active Holdtime: 90
  Keepalive Interval: 30         Group index: 0    Peer index: 0    SNMP index: 10    
  I/O Session Thread: bgpio-0 State: Enabled
  BFD: disabled, down
  Local Interface: ge-0/0/1.0                       
  NLRI for restart configured on peer: inet-unicast
  NLRI advertised by peer: inet-unicast inet6-unicast
  NLRI for this session: inet-unicast
  Peer supports Refresh capability (2)
  Stale routes from peer are kept for: 300
  Peer does not support Restarter functionality
  Restart flag received from the peer: Notification
  NLRI that restart is negotiated for: inet-unicast
  NLRI of received end-of-rib markers: inet-unicast
  NLRI of all end-of-rib markers sent: inet-unicast
  Peer does not support LLGR Restarter functionality
  Peer supports 4 byte AS extension (peer-as 65001)
  Peer does not support Addpath
  NLRI that we support extended nexthop encoding for: inet-unicast
  NLRI that peer supports extended nexthop encoding for: inet-unicast
  NLRI(s) enabled for color nexthop resolution: inet-unicast
  Table inet.0 Bit: 20000
    RIB State: BGP restart is complete
    Send state: in sync                 
    Active prefixes:              1
    Received prefixes:            1
    Accepted prefixes:            1
    Suppressed due to damping:    0
    Advertised prefixes:          2
  Last traffic (seconds): Received 15   Sent 25   Checked 488 
  Input messages:  Total 22     Updates 2       Refreshes 0     Octets 554
  Output messages: Total 21     Updates 2       Refreshes 0     Octets 540
  Output Queue[1]: 0            (inet.0, inet-unicast)

[edit]

root@CORE-1# run show route receive-protocol bgp fe80::e20:51ff:fed2:1%ge-0/0/1.0 
inet.0: 3 destinations, 3 routes (3 active, 0 holddown, 0 hidden)
Limit/Threshold: 1048576/1048576 destinations
  Prefix                  Nexthop              MED     Lclpref    AS path
* 10.1.1.2/32             fe80::e20:51ff:fed2:1                   65001 I

inet6.0: 3 destinations, 3 routes (3 active, 0 holddown, 0 hidden)
Limit/Threshold: 1048576/1048576 destinations

```

IPv4 prefix is visible via IPv6 next-hop and it is also reachable:

```

root@CORE-1# run ping 10.1.1.2 source 10.1.1.1 count 1     
PING 10.1.1.2 (10.1.1.2): 56 data bytes
64 bytes from 10.1.1.2: icmp_seq=0 ttl=64 time=2.858 ms

--- 10.1.1.2 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max/stddev = 2.858/2.858/2.858/0.000 ms

[edit]
root@CORE-1# 

```

## Word on avoiding accidents ##

It seems that dynamic-neighbor cannot be configured with authentication algorithm:

```
root@ACCESS-1# show groups       
BGP-AUTH {
    protocols {
        bgp {
            group <*> {
                authentication-algorithm aes-128-cmac-96;
                authentication-key-chain BGP-AUTH;
            }
        }
    }
}

[edit]
root@ACCESS-1# show security authentication-key-chains 
key-chain BGP-AUTH {
    tolerance 600;
    key 0 {
        secret "$9$OA0xIRcMWXx-bcSVs2gZG/9A0RcM8xs2oz3Cp"; ## SECRET-DATA
        start-time "2023-11-29.06:01:00 +0000";
    }
}

[edit]
root@ACCESS-1# set protocols bgp group UNDERLAY apply-groups BGP-AUTH 

[edit]
root@ACCESS-1# commit 
[edit protocols bgp group UNDERLAY dynamic-neighbor bgp_unnumbered]
  'peer-auto-discovery'
    Can't be configured along with authentication-algorithm
[edit protocols]
  'bgp'
    warning: requires 'bgp' license
error: commit failed: (statements constraint check failed)

[edit]
root@ACCESS-1# 

```

Same is true for authentication-key. So what can one do here to avoid accidents, and to which extent is this a risk?

On one hand, we're only using link-local addresses and assuming that the links are physically p2p between two devices the risk is mitigated to a point.
This is especially true since we have not configured any routable addressing for links over which the auto discovery works.

It is trivial to create a firewall filter to limit the access to tcp/179 but this makes the whole concept of dynamic-neighbor bit questionable as the addresses would have to be preconfigured somewhere.

Having authentication would be handy as its possible to preprovision this without much of a hassle but either I haven't figured out how this works in this case or it is not possible at the moment.




