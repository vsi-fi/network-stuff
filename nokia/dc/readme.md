# Implementing basic data centre routing to test containerlab and SRLinux #
I wanted to try building a network with some basic data centre functions using Nokia SRLinux.
I set myself the following "goals":
* Network should have no obvious single point of failures
* Links between network devices should be included in ECMP scheme
* Each "site" in the network would interconnect using eBGP
* See if we can implement active-active multi homing towards hosts (LACP)
* cross-site DHCP relay
* Use different IGPs in least two of the sites
* Implement anycast-gw
* Implement anycast routing where "service address" is advertised from least two sites
* Attempt using 32 bit AS numbers internally and 16 bit for outside world, try overwriting the 32 bit ASs from the advertisement

All in all, this is still very much a work-in-progress.

## Description of the topology ##
We have one "main" hub site that attaches to to the outside world using BGP.
This site runs OSPF as its IGP and hosts a pair of access devices that produce active-active multi homing for a test host.
As this site receives the default route from outside world it is also injecting the default to its IGP and via BGP to other sites for further distribution.
## Description of tooling ##
For this experiment the logical choice seemed to be the [containerlab](https://containerlab.dev/) which allows one to run "containerised" images of network operating systems and to an extent their forwarding planes.
While very nice to work with, it also comes with some caveats such as not being able to actually implement class-of-service in some instances etc.
## What is container lab? ##
"Containerlab provides a CLI for orchestrating and managing container-based networking labs. It starts the containers, builds a virtual wiring between them to create lab topologies of users choice and manages labs lifecycle." (from containerlab website).
### Spec file ###
Containerlab parses its specification from a YAML file. I do not claim to have any expertise on any of this but the below file is used as template for this test.
```
cat NS.clab.yml
# topology documentation: http://containerlab.dev/lab-examples/srl-ceos/
name: DC

topology:
  nodes:
    site1-core-1:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    site1-core-2:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    site3-core-1-1:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    site3-core-1-2:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    site2-core-1-1:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    site2-core-1-2:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    #Access devices, mh for multihoming
    sw-data-tpu-1:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    sw-data-mh-1-1:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    sw-data-mh-1-2:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    sw-data-site2-1:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    sw-data-site3-1-1:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    #Outside routers, these should receive the aggregate routes from the site1 cores
    outside-it-rtr-1:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    outside-it-rtr-2:
      kind: nokia_srlinux
      image: ghcr.io/nokia/srlinux:24.10
    #Hosts
    site1-tpu-1-1:
      kind: linux
      image: labalpine
    site1-mh-host-1:
      kind: linux
      image: labalpine
    site3-host-1:
      kind: linux
      image: labalpine
    site2-host-1:
      kind: linux
      image: labalpine
    dhcp-1:
      kind: linux
      image: labalpine

  links:
    #core links
    - endpoints: ["site1-core-1:e1-10", "site1-core-2:e1-10"]
    - endpoints: ["site2-core-1-1:e1-10","site2-core-1-2:e1-10"]
    - endpoints: ["site3-core-1-1:e1-10", "site3-core-1-2:e1-10"]
    - endpoints: ["site1-core-1:e1-11", "site3-core-1-1:e1-11"]
    - endpoints: ["site1-core-2:e1-11", "site3-core-1-2:e1-11"]
    - endpoints: ["site1-core-1:e1-12", "site2-core-1-1:e1-11"]
    - endpoints: ["site1-core-2:e1-12", "site2-core-1-2:e1-11"]
    - endpoints: ["site1-core-1:e1-16", "outside-it-rtr-1:e1-1"]
    - endpoints: ["site1-core-2:e1-16", "outside-it-rtr-1:e1-2"]


    #aggregation links
    - endpoints: ["site1-core-1:e1-1", "sw-data-tpu-1:e1-1"]
    - endpoints: ["site1-core-2:e1-1", "sw-data-tpu-1:e1-2"]

    - endpoints: ["site1-core-1:e1-2", "sw-data-mh-1-1:e1-1"]
    - endpoints: ["site1-core-2:e1-2", "sw-data-mh-1-1:e1-2"]
    - endpoints: ["site1-core-1:e1-3", "sw-data-mh-1-2:e1-1"]
    - endpoints: ["site1-core-2:e1-3", "sw-data-mh-1-2:e1-2"]

    - endpoints: ["site2-core-1-1:e1-1", "sw-data-site2-1:e1-1"]
    - endpoints: ["site2-core-1-2:e1-1", "sw-data-site2-1:e1-2"]
    - endpoints: ["site3-core-1-1:e1-1", "sw-data-site3-1-1:e1-1"]
    - endpoints: ["site3-core-1-2:e1-1", "sw-data-site3-1-1:e1-2"]

    #access links
    - endpoints: ["sw-data-tpu-1:e1-10", "site1-tpu-1-1:eth1"]
    - endpoints: ["sw-data-mh-1-1:e1-10", "site1-mh-host-1:eth1"]
    - endpoints: ["sw-data-mh-1-2:e1-10", "site1-mh-host-1:eth2"]
    - endpoints: ["sw-data-site3-1-1:e1-10", "site3-host-1:eth1"]
    - endpoints: ["sw-data-site2-1:e1-10", "site2-host-1:eth1"]
    - endpoints: ["site1-core-1:e1-17", "dhcp-1:eth1"]
    - endpoints: ["site1-core-2:e1-17", "dhcp-1:eth2"]
```
### Starting / stopping containerlab ###
Lab can be started from the directory where the spec file is located:
```
pwd
~/lab_directory
ls -l NS.clab.yml
 ls -l NS.clab.yml
-rw-r--r-- 1 username username 3389 Nov 26 08:33 NS.clab.yml
sudo containerlab deploy
```
Notice that sudo is required for reasons such as containerlab populating /etc/hosts -file etc.
To stop the lab run "sudo containerlab destroy". Unlike what you might think from the "destroy" it appears to keep the config files intact if you have configured SRLinux to save its config. More on this quirky behaviour later.
In my test the lab took about 21gigs of ram to run which is still fairly reasonable considering the number of instances involved.
## Configuration samples ##
I'll try to cover the possible interesting pieces of the configuration next, especially the parts that seemed little bit different compared to Junos or otherwise foreign to me. 
### Alpine linux container ###
In this lab i also wanted to use a slightly modified [alpine linux](https://www.alpinelinux.org/) with some extra packages included.
This alpine instance makes it little bit easier to debug/test the functionality of the network.
Docker images are built from a cleverly named Dockerfile:
```
cat Dockerfile
FROM alpine:latest

RUN apk update && apk add --no-cache \
	openssh \
	kea-dhcp4 \
	dhcpcd \
	sudo \
	bash \
	bird \
	curl \
	nmap \
	vlan \
	bonding \
	tcpdump \
	bird \
	vim

RUN adduser -D labuser && echo "labuser:password" | chpasswd
RUN echo "root:password" | chpasswd
RUN ssh-keygen -A
RUN echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
RUN echo "PasswordAuthentication yes" >> /etc/ssh/sshd_config

EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
```
In the above file we take the base alpine image, shove some useful packages to it, create a user labuser with password "password" and set the same password for root.
We also allow root to login via ssh and "expose" the port 22 for ssh.
The image itself can be built using "docker build --no-cache -t labalpine" no-cache here is useful if you need to rebuild the image several times "from scratch".
Once the image is built, you can test it out with:
```
docker images|grep -P 'REPOSITO|labal'
REPOSITORY      TAG       IMAGE ID       CREATED              SIZE
labalpine       latest    e290439acbb1   About a minute ago   89.5MB

docker run -d -p 2222:22 --name alpine-test labalpine
dfd709a6864369ce18246a48e066b908d9d94761d4efe2b3e0d9402d4bb65471

ssh -p 2222 localhost -l root
The authenticity of host '[localhost]:2222 ([127.0.0.1]:2222)' can't be established.
ED25519 key fingerprint is: SHA256:z2MaLOOu81LGVHSzBzEOJYQG3LvphyF2I4v+ifHSNrA
This key is not known by any other names.
Are you sure you want to continue connecting (yes/no/[fingerprint])? yes
Warning: Permanently added '[localhost]:2222' (ED25519) to the list of known hosts.
root@localhost's password:
Welcome to Alpine!

The Alpine Wiki contains a large amount of how-to guides and general
information about administrating Alpine systems.
See <https://wiki.alpinelinux.org/>.

You can setup the system with the command: setup-alpine

You may change this message by editing /etc/motd.

dfd709a68643:~#
```
The above implies that you've successfully built the image and is available to you, you can start it and ssh to it.
However, this is not really the method how containerlab uses the said image but above is still useful to understand/verify that the thing works as we want.
You can run "docker ps" to see what images are running and stop the image with "docker stop":
```
$ docker ps|grep -P 'CONTAINER|labalpine'
CONTAINER ID   IMAGE                  COMMAND               CREATED         STATUS                 PORTS                                                                              NAMES
dfd709a68643   labalpine              "/usr/sbin/sshd -D"   3 minutes ago   Up 3 minutes           0.0.0.0:2222->22/tcp                                                               alpine-test
$ docker stop dfd709a68643
dfd709a68643
```
After this you should no longer see the alpine as a running image (docker ps)
### Basic SRLinux setup ###
I do not know why, but it seems that least 7220 IXR-D2L / v24.10.5 need the following pieces in order to automatically write the config file at commit time and to create a "checkpoint" into which one can rollback to:
```
--{ candidate shared default }--[  ]--
A:outside-it-rtr-1# / set system configuration auto-checkpoint true

--{ * candidate shared default }--[  ]--
A:outside-it-rtr-1# / set system configuration auto-save true

--{ * candidate shared default }--[  ]--
A:outside-it-rtr-1# diff
      system {
          configuration {
+             auto-checkpoint true
+             auto-save true
          }
      }

--{ * candidate shared default }--[  ]--
A:outside-it-rtr-1# commit stay
/system:
    Generated checkpoint '/etc/opt/srlinux/checkpoint/checkpoint-0.json' with name 'checkpoint-2025-11-26T08:31:30.356Z' and comment 'automatic checkpoint after commit 3'

/system:
    Saved current running configuration as initial (startup) configuration '/etc/opt/srlinux/config.json'

All changes have been committed. Starting new transaction.

--{ candidate shared default }--[  ]--
```
After this you can do stuff like "load checkpoint id 1" which is functionally very close to "rollback 1" in Junos for instance. Very handy.
### BGP setup on the "outside" world routers ###
Rest of the world is supposed to be reachable from an ISP whose AS number is 64496.
This ISP is exporting default route (0.0.0.0/0) to us and accepting the aggregate we're using for our DC (10.0.0.0/8).
We have a pair of links between the ISP routers and the site1-cores, 172.16.0.0/31 and 172.16.0.2/31.

#### Default route ####
In order to have a default route to advertise we need to create one:
```
A:outside-it-rtr-1# info network-instance WORLD static-routes
    network-instance WORLD {
        static-routes {
            route 0.0.0.0/0 {
                next-hop-group BLACKHOLE
            }
        }
    }

--{ * candidate shared default }--[  ]--
A:outside-it-rtr-1# info network-instance WORLD next-hop-groups group BLACKHOLE
    network-instance WORLD {
        next-hop-groups {
            group BLACKHOLE {
                blackhole {
                }
            }
        }
    }
```
The above effectively creates a nice little singularity for us to offer to our unsuspecting customers.
You can view the instance-specific routing table to verify:
```
A:outside-it-rtr-1# show network-instance WORLD route-table ipv4-unicast route 1.2.3.4 |grep -P 'WORLD|Route Type'
IPv4 unicast route table of network instance WORLD
|                      Prefix                       |  ID   | Route Type |     Route Owner      |  Active  |  Origin  | Metric  |    Pref    |        Next-hop (Type)        |      Next-hop Interface       |    Backup Next-hop (Type)     |                          Backup Next-hop Interface                          |
| 0.0.0.0/0                                         | 0     | static     | static_route_mgr     | True     | WORLD    | 1       | 5          | blackholed [discard]          | blackholed                    |                               |                                                                             |

--{ candidate shared default }--[  ]--
```
So the blackhole is firmly installed. For our customer to get a chance of enjoying this gravity bending product we need to apply an export policy, alongside of an import policy that matches our customers network:
```
A:outside-it-rtr-1# info routing-policy
    routing-policy {
        prefix-set DEFAULT {
            prefix 0.0.0.0/0 mask-length-range exact {
            }
        }
        prefix-set IPV4-IMPORT-CUSTOMER-64496 {
            prefix 10.0.0.0/8 mask-length-range exact {
            }
        }
        policy IPV4-EXPORT-CUSTOMER-64496 {
            default-action {
                policy-result reject
            }
            statement DEFAULT {
                match {
                    prefix-set DEFAULT
                }
                action {
                    policy-result accept
                }
            }
        }
        policy IPV4-IMPORT-CUSTOMER-64496 {
            default-action {
                policy-result reject
            }
            statement IPV4-IMPORT-CUSTOMER-64496 {
                match {
                    prefix-set IPV4-IMPORT-CUSTOMER-64496
                    protocol bgp
                }
                action {
                    policy-result accept
                }
            }
        }
    }

--{ candidate shared default }--[  ]--
```
This is functionally quite obvious and very similar to how Junos implements routing policy. Also, identically to Junos, the policy is applied at the bgp group -level.
Similarly, customer side has a policy matching our specs:
```
A:site1-core-1# info routing-policy
    routing-policy {
        prefix-set DC {
            prefix 10.0.0.0/8 mask-length-range exact {
            }
        }
        prefix-set DEFAULT {
            prefix 0.0.0.0/0 mask-length-range exact {
            }
        }
        policy IPV4-EXPORT-DC {
            default-action {
                policy-result reject
            }
            statement DC {
                match {
                    prefix-set DC
                }
                action {
                    policy-result accept
                }
            }
        }
        policy IPV4-IMPORT-ISP1 {
            default-action {
                policy-result reject
            }
            statement DEFAULT {
                match {
                    prefix-set DEFAULT
                    protocol bgp
                }
                action {
                    policy-result accept
                }
            }
        }
    }

--{ + candidate shared default }--[  ]--
``` 
And if we look at the routing tables on both sides we can see the expected routes:
```
A:site1-core-1# show network-instance SITE-1 route-table ipv4-unicast route 1.2.3.4 | grep SITE
IPv4 unicast route table of network instance SITE-1
| 0.0.0.0/0                            | 0     | bgp        | bgp_mgr              | True     | SITE-1   | 0       | 170        | 172.16.0.0/31          | ethernet-1/16.100      |                        |                                               |

--{ + candidate shared default }--[  ]--

A:outside-it-rtr-1# show network-instance WORLD route-table ipv4-unicast route 10.1.2.3|grep WORL
IPv4 unicast route table of network instance WORLD
| 10.0.0.0/8                           | 0     | bgp        | bgp_mgr              | True     | WORLD    | 0       | 170        | 172.16.0.0/31          | ethernet-1/1.100       |                        |                                               |

--{ candidate shared default }--[  ]--
```
If we enable the other bgp session from site1-core-2 to outside-it-rtr-1 the session comes up and we can see that both neighbours are sending the same 10.0.0.0/8.
However, we want all links active so we need to enable something similar to multipathing in Junos:
```
A:outside-it-rtr-1# / show network-instance WORLD route-table |grep -A1 10.0.0.0/8
| 10.0.0.0/8                           | 0     | bgp        | bgp_mgr              | True     | WORLD    | 0       | 170        | 172.16.0.0/31          | ethernet-1/1.100       |                        |                                               |
|                                      |       |            |                      |          |          |         |            | (indirect/local)       |                        |   
A:outside-it-rtr-1# diff
      network-instance WORLD {
          protocols {
              bgp {
                afi-safi ipv4-unicast {
                    + multipath {
                        + maximum-paths 8
                    + }
                  }
              }
          }
      }
commit stay
-{ candidate shared default }--[ network-instance WORLD protocols bgp ]--
A:outside-it-rtr-1# / show network-instance WORLD route-table | grep -A3 10.0.0.0
| 10.0.0.0/8                           | 0     | bgp        | bgp_mgr              | True     | WORLD    | 0       | 170        | 172.16.0.0/31          | ethernet-1/1.100       |                        |                                               |
|                                      |       |            |                      |          |          |         |            | (indirect/local)       | ethernet-1/2.100       |                        |                                               |
|                                      |       |            |                      |          |          |         |            | 172.16.0.2/31          |                        |                        |                                               |
|                                      |       |            |                      |          |          |         |            | (indirect/local)       |                        |                        |                                               |
--{ candidate shared default }--[ network-instance WORLD protocols bgp ]
```
This effectively means that the same prefix is now available via two paths over two physical connections

#### IGP and iBGP arrangements in the DC core ####
There is also a internal BGP session between the DC SITE-1 core routers. This session can be used to keep a valid default route(s) in the core devices in case one of them looses its BGP session with the ISP-1 router. However, for this to work there are few different implementation options:
* Inject default route as OSPF external to the area 0
* Create static default with bad metric to be used as backup route
* Advertise the default via iBGP
Lets say core-1 were to drop its eBGP session with ISP-1 and we'd have the iBGP advertisement working.
It happens that the next-hop for the default route via core-2 points to ISP-1 router interface. This likely means that core-1 cannot resolve the next-hop.
To work around this we could:
* Add the link towards ISP-1 as passive in ospf area 0
* Create export policy that advertises the directly attached networks to ospf as externals
* Use nexthop-self for the iBGP sessions
Last one of the three seems least tempting to me. Reasoning being that if at some later date one would like to use the cores as route-reflectors they could potentially introduce sub-optimal routing.
This is especially true as I am hoping to configure EVPN for multihoming hosts to two access devices. For this route-reflectors at core seem reasonable.
I opted to configure the links into area 0 as passive, mainly due to laziness as this way i didn't need to write and apply the export policy. 
```
A:site1-core-2# info network-instance SITE-1 protocols ospf
    network-instance SITE-1 {
        protocols {
            ospf {
                instance SITE-1 {
                    admin-state enable
                    version ospf-v2
                    router-id 10.255.255.2
                    area 0.0.0.0 {
                        interface ethernet-1/10.100 {
                            interface-type point-to-point
                        }
                        interface ethernet-1/16.100 {
                            passive true
                        }
                        interface lo0.1 {
                            passive true
                        }
                    }
                }
            }
        }
    }
```
Here we simply stablish the IGP session over eth-1/10.100 and add the loopback for iBGP and eth-1/16.100 for reachability to ISP-1.
BGP looks something like this:
```
A:site1-core-2# info network-instance SITE-1 protocols bgp
    network-instance SITE-1 {
        protocols {
            bgp {
                autonomous-system 64497
                router-id 10.255.255.2
                afi-safi ipv4-unicast {
                    admin-state enable
                }
                group ISP-1 {
                    peer-as 64496
                    export-policy [
                        IPV4-EXPORT-DC
                    ]
                    import-policy [
                        IPV4-IMPORT-ISP1
                    ]
                }
                group SITE-1-IBGP {
                    peer-as 64497
                    afi-safi ipv4-unicast {
                        admin-state enable
                    }
                    transport {
                        local-address 10.255.255.2
                    }
                }
                neighbor 10.255.255.1 {
                    peer-group SITE-1-IBGP
                }
                neighbor 172.16.0.2 {
                    peer-group ISP-1
                }
            }
        }
    }
--{ + candidate shared default }--[  ]--
```

## Tests (to be done) ##
### Test failovers for the default routes ###
### Test failovers for EVPN-ESIs ###
### Test that dhcp-relay works ###
