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

#### Route reflector ####

### PE routers ###
