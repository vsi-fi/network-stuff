# Underlay network using IS-IS for loopback connectivity #

I'd like to start by stating that there are many methods of building functional underlay network for a EVPN deployment.
In this setup I set the below goals for myself:

* Simple to setup with small number of 'per interface' configurables.
* Produce redundant, active-active paths between the loopbacks.
* Use authentication.
* Advertise only the loopbacks.
* Allow for 'plug and play' deployment.

I decided to use IS-IS as my interior gateway protocol of choice here as it seemed like the most straight forward option.
I would have also been possible to configure eBGP based underlay lay but as I was doing this by hand I wanted to cut down on the amount of configuration. 

What follows is the 'most interesting' configuration sections of the IS-IS based underlay.

### Interfaces ###
We need to configure couple of addresses for our lab to get the underlay working.

**Loopback interface**

* IP address. This will later on be used as our virtual tunnel endpoint address (VTEP) 
* ISO connectionless network protocol(CLNP) address family required by IS-IS this is used to network service access point, or NSAP used to identify a point of connection to the network.

The set commands are as follows:

    set interfaces lo0 unit 0 family inet address 10.1.1.1/32
    set interfaces lo0 unit 0 family iso address 49.0002.0010.0001.0001.0001.00

This in turn results in the below config:

    root@SITE-1-SPINE-1-A> show configuration interfaces lo0    
    unit 0 {
        family inet {
            address 10.1.1.1/32;
        }
        family iso {
            address 49.0002.0010.0001.0001.0001.00;
        }
    }

Couple of additional details concerning the above NSAP address:

* *49* is the authority format identifier, 49 stands for "private assignment", maybe bit similar to RFC1918 IP addresses.
* *0002* is the area identifier, this could have been used to split a routing domain to subdomains.
* *0010.0001.0001.0001* this is the "system identifier". I've taken this from the IPv4 address of the same loopback. 
* *00* stands for "selector" or NSEL, this would have been used to identify different "services" provided by the router. 

I believe that the above complexities are derived from the original plans of using ISO addresses as globally routable, similar to IP.
My understanding is that the ISO addresses are almost exclusively used for interior gateway protocol IS-IS and maybe for some mobile network applications.

**Physical interfaces**

Instead of individually numbering the physical interfaces I am "loaning" the IPv4 address of the lo0.0.
This allows for bit less things to document as the individual point-to-point links between the devices do not need specific subnet allocations and also allows for "plug-and-play" type of setup.

    set interfaces ge-0/0/0 unit 0 family inet unnumbered-address lo0.0
    set interfaces ge-0/0/0 unit 0 family iso
    set interfaces ge-0/0/1 unit 0 family inet unnumbered-address lo0.0
    set interfaces ge-0/0/1 unit 0 family iso
    set interfaces ge-0/0/2 unit 0 family inet unnumbered-address lo0.0
    set interfaces ge-0/0/2 unit 0 family iso

The above results in:

    root@SITE-1-SPINE-1-A> show configuration interfaces 
    ge-0/0/0 {
        unit 0 {
            family inet {
                unnumbered-address lo0.0;
            }
            family iso;
        }
    }
    ge-0/0/1 {
        unit 0 {
            family inet {
                unnumbered-address lo0.0;
            }
            family iso;
        }
    }
    ge-0/0/2 {
        unit 0 {
            family inet {
                unnumbered-address lo0.0;
            }
            family iso;
        }
    }                                       

Note that we also have to enable the ISO address family on the physical interfaces in order to run IS-IS over the said interfaces.

### Routing options ###

We want to set the router id on each device. This should be a unique address.
Router id is used to identify the device from which the routing information originated.
It should be noted that if router id is not configured, it tends to be auto derived from the first interface that came up as part of the boot process. Quite often this turns out to be the loopback.
In this example I am picking the address from the loopback interface

    set routing-options router-id 10.1.1.1
    set routing-options forwarding-table export ECMP 

This results in:

    root@SITE-1-SPINE-1-A> show configuration routing-options       
    router-id 10.1.1.1;
    forwarding-table {
        export ECMP;
    }

In the above snippet we set the same address as router id as to what we had previously configured as IPv4 address for interface lo0 unit 0. We're also applying an export policy to forwarding table that tells the device to do "per packet" load balancing. Juniper has bit of a caveat related to this as the exact behavior of the "per-packet" load balancing depends on the hardware platform. However, in the case of QFX this doesn't really mean "per-packet" but more like "per-flow" or stream of traffic. 

Flows can be identified by looking at the below attributes of traffic:

* source/destination address
* protocol
* source/destination port if applicable
* type of service field in the packet (ToS)
* interface index which is used to numerically identify individual interfaces ( for example ge-0/0/0 )

Anyway, the load balancing policy is configured as follows:

    set policy-options policy-statement ECMP then load-balance per-packet
    root@SITE-1-SPINE-1-A> show configuration policy-options policy-statement ECMP
    then {
        load-balance per-packet;
    }

 
### IS-IS - some background and reasoning ###

I am using IS-IS as the routing protocol of choice to generate reachability between the loopback interfaces. Below are some ramblings related to the IS-IS protocol and how it compares to the other popular interior gateway protocol, OSPF.

IS-IS or intermediate system to intermediate system is a link state interior gateway protocol or IGP, it shares certain commonalities with OSPF or open shortest path first, among which are at least the following:

* Reliable flooding of link state information amongst the participating routers
* Dijkstra's algorithm is used to compute the "best" path through the network.
* Said algorithm is ran against the link state database.
* Both are hierarchical but the implementations differ as "areas" have a different meaning and IS-IS has the concept of levels. 
* In case of a broadcast medium, both have the concept of a designated system or router that removes the need for complete full-mesh adjacencies. 
* Both can use authentication.
* Unnumbered interfaces are supported on both.

The above are relevant to our use case, but there are other common aspects such as support for variable length subnet masks etc.

However, it is worth noting that there are rather significant dissimilarities as well:

* IS-IS is a OSI layer two protocol, whereas OSPF runs on IP.
* IS-IS uses OSI addressing.
* IS-IS does not have a backup designated intermediate system whereas OSPF has backup designated router. 
* IS-IS doesn't run the shortest path calculation when a network prefix goes down, but when the status of a intermediate system (router) changes. To put this other words: SPF is ran to determine the "best" path to a router, not to the prefixes mentioned in the link state advertisements. Partial routing table calculation is used to generate the routing table. 
* IS-IS doesn't do periodic database refresh whereas OSPF (by default does).
* IS-IS has the concept of levels - this produces part of the hierarchy component:
    * Level 2 forms the backbone of an IS-IS network
        * Level 2 routers form adjacencies only with other level 2 routers and L1/L2 routers (area border routers)
    * Level 1 forms a "normal area", e.g. non-backbone area
        * Level 1 routers form adjacencies with only other level 1 routers or L1/L2 routers (area border routers) 
* IS-IS actually detects the name of the adjacent router on the 'application level' whereas OSPF picks up only IP.

In my subjective opinion the hassle with the ISO addressing is compensated by the less chatty nature of IS-IS compared to OSPF. IS-IS feels also simpler to configure and there is no not-so-stub-areas etc. 
IS-IS can also carry more than one address family in the same adjacency. For example, this means that it is possible to have both IPv4 and IPv6 handled by the same adjacency and you do not need separate processes of OSPFv2 and OSPFv3.
However, in the case of Juniper IS-IS requires more costly license. Yet, the same expensive license is required for BGP and EVPN so it is bit of a moot point. 
 
### IS-IS & BFD - configuration ###

Below is the IS-IS configuration for the SPINE-1-A

    root@SITE-1-SPINE-1-A> show configuration protocols isis   
    apply-groups [ IS-IS-BFD IS-IS-AUTHENTICATION ];
    interface ge-0/0/0.0 {
        point-to-point;
    }
    interface ge-0/0/1.0 {
        point-to-point;
    }
    interface ge-0/0/2.0 {
        point-to-point;
    }
    interface lo0.0 {
        passive;
    }
    level 2 wide-metrics-only;

Starting from the loopback interface, we're including it as a "passive" interface, meaning that we only want to announce the address space configured on it, not form adjacencies over it. In general, it is maybe a good idea to configure all access device facing ports as passive as well.

Physical interfaces are added as point-to-point. This p2p converts the broadcast ethernet medium to be considered as p2p. This has at least the following impacts:

* No link specific addressing is required
* On a p2p link, each link state protocol data unit (PDU) is acknowledged by a partial sequence number PDU whereas on a broadcast medium a complete sequence number PDU is sent.
* There is no meaningful concept of designated IS on a point-to-point link, similar to OSPF not having to negotiate a designated router. 

My unscientific opinion is that the point-to-point should converge tiny bit faster and is also simpler to configure.

Wide-metrics-only tells IS-IS to generate metric values greater than 63 on level 1/2 which the only level used in our lab.
This is strictly speaking not required for our lab but something that allows for larger networks and is maybe more aligned with the modern times. 

The apply-groups statement provisions bidirectional forward detection (BFD) on the IS-IS adjacencies. BFD is a "light weight" helo protocol that can quickly detect if the path between the devices is down or if there is some "lights are on but nobody is at home" type of situation. Also, this allows quicker experimentation with failovers since in my virtual lab the link pausing doesn't seem to reliably relay the link-down events to the devices.

BFD config looks as follows:

    root@SITE-1-SPINE-1-A# show groups IS-IS-BFD 
    protocols {
        isis {
            interface <ge-0/0/*> {
                family inet {
                    bfd-liveness-detection {
                        minimum-interval 1000;
                        multiplier 4;
                    }
                }
            }
        }
    }

Unlike IS-IS, BFD is a IP protocol, hence we need to explicitly state that the address family to be used is inet.

**Minimum-interval** is the interval (in milliseconds) at which the device sends hello packets and expects to receive replies.

**Multiplier** is the number of helo packets without replies the device tolerates before it considers the adjacent device dead.

IS-IS also supports authentication which is per level. Authentication is configured via the apply group as follows:

    root@SITE-1-SPINE-1-A> show configuration groups IS-IS-AUTHENTICATION 
    protocols {
        isis {
            level 1 {
                /* Password: IS-IS-PASSWORD */
                authentication-key "$9$r28eMLNdwoJGP5RcyKW8aJGDjkP5z6Cp7-DH.mzF9Ct"; ## SECRET-DATA
                authentication-type md5;
            }
        }
    }

    root@SITE-1-SPINE-1-A> 

This configuration authenticates the protocol exchanges as an attempt to minimise the risk of accidentally forming adjacencies with wrong devices. If the password is wrong or missing the adjacency fails to come up and is stuck at "Initializing" phase. 

### IS-IS & BFD - verification ###

We'll start by verifying the status of BFD

    root@SITE-1-SPINE-1-A> show bfd session 
                                                      Detect   Transmit
    Address                  State     Interface      Time     Interval  Multiplier
    10.1.10.1                Up        ge-0/0/1.0     4.000     1.000        4   
    10.1.10.2                Down      ge-0/0/2.0     0.000     2.000        4   

    2 sessions, 2 clients
    Cumulative transmit rate 1.5 pps, cumulative receive rate 1.0 pps

Here we can see that we have two BFD sessions configured but the one going to the LEAF-1-B is down.
However, the IS-IS adjacency itself is up:

    root@SITE-1-SPINE-1-A> show isis adjacency 

    Warning: License key missing; requires 'isis' license

    Interface             System         L State         Hold (secs) SNPA
    ge-0/0/1.0            SITE-1-LEAF-1-A 3 Up                    26
    ge-0/0/2.0            SITE-1-LEAF-1-B 3 Up                    20

    root@SITE-1-SPINE-1-A> 

Reason for this is that the apply group that provisions BFD for IS-IS is not applied on LEAF-1-B.
This is to illustrate that BFD being down, doesn't mean that the routing protocol has to be down, it is just an additional safe guard that can detect failure quicker than the helo packets used by the protocol. 

Lets take a quick peek at the routes received over IS-IS

    root@SITE-1-SPINE-1-A> show route protocol isis table inet.0 

    Warning: License key missing; requires 'isis' license

    inet.0: 4 destinations, 4 routes (4 active, 0 holddown, 0 hidden)
    Limit/Threshold: 1048576/1048576 destinations
    + = Active Route, - = Last Active, * = Both

    10.1.1.2/32        *[IS-IS/15] 00:05:39, metric 20
                          to 10.1.10.1 via ge-0/0/1.0
                       >  to 10.1.10.2 via ge-0/0/2.0
    10.1.10.1/32       *[IS-IS/15] 00:16:39, metric 10
                       >  to 10.1.10.1 via ge-0/0/1.0
    10.1.10.2/32       *[IS-IS/15] 00:05:39, metric 10
                        >  to 10.1.10.2 via ge-0/0/2.0

    root@SITE-1-SPINE-1-A> 

From here we can see that we've learned of the loopbacks of both of the leaf switches (10.1.10.1 and 10.1.10.2).
Also, we have two routes to 10.1.1.2 which is the loopback of the SITE-1-SPINE-1-B.
However, the route 10.1.1.2 is bit more interesting than the routes to the leaf switches:

    root@SITE-1-SPINE-1-A> show route forwarding-table destination 10.1.1.2/32 table default
    Routing table: default.inet
    Internet:
    Destination        Type RtRef Next hop           Type Index    NhRef Netif
    10.1.1.2/32        user     0                    ulst  1048574     2
                                  10.1.10.1          ucst      582     4 ge-0/0/1.0
                                  10.1.10.2          ucst      583     5 ge-0/0/2.0
 
This goes to say that the equal cost multi pathing is working as defined in the routing-options section.

We have two active paths of similar cost, so why not use both. ECMP comes to its own when we start sending traffic over VXLAN tunnels later on. 


Finally, we can run a quick ping between the leaf switches to verify reachability:

    root@SITE-1-LEAF-1-A> ping 10.1.10.2 source 10.1.10.1 count 1 
    PING 10.1.10.2 (10.1.10.2): 56 data bytes
    64 bytes from 10.1.10.2: icmp_seq=0 ttl=63 time=6.623 ms

    --- 10.1.10.2 ping statistics ---
    1 packets transmitted, 1 packets received, 0% packet loss
    round-trip min/avg/max/stddev = 6.623/6.623/6.623/0.000 ms

    root@SITE-1-LEAF-1-A> 

**At this stage the underlay is pretty much ready to go.**


