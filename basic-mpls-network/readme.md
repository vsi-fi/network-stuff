# Trivial MPLS network #

Idea with this lab is to build a simple MPLS network in order to test different EVPN data centre interconnection options across a service provider -type network.

In order to keep things simple I opted for the below implementation specs for the SP network:

* IS-IS as IGP
* IBGP with route-reflector on one of the P routers
* LDP instead of RSVP for initial setup
* L2vpn interconnection between sites
* L3vpn interconnection between sites
* Possibly service vrf inside SP network that could be made available in the L3 DCI vrf

## Building blocks ##

I used the below "equipment":

* vmx 21.4R1.12 for all SP gear

## Configuration ##

### P routers ###

Starting with the interface configuration:

```
root@ISP-2-P> show configuration interfaces 
ge-0/0/0 {
    description "Connection to ISP-1-P";
    mtu 9500;
    unit 0 {
        family inet {
            unnumbered-address lo0.0;
        }
        family iso;
        family mpls;
    }
}
ge-0/0/1 {
    description "Connection to ISP-3-P";
    mtu 9500;
    unit 0 {
        family inet {
            unnumbered-address lo0.0;
        }
        family iso;
        family mpls;
    }
}
ge-0/0/9 {
    description "Connection to ISP-PE-2";
    mtu 9500;
    unit 0 {
        family inet {
            unnumbered-address lo0.0;
        }
        family iso;
        family mpls;
    }
}
lo0 {
    unit 0 {
        family inet {
            address 10.0.0.2/32;
        }
        family iso {
            address 49.0002.0010.0000.0000.0002.00;
        }
    }
}

```

#### Route reflector ####

### PE routers ###
