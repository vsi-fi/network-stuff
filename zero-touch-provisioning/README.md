# Trivial example of how to utilise zero touch provisioning #
So called zero touch provisioning allows for 'quicker' deployment of Juniper devices.

When a new Junos box is installed it tries to run DHCP client once it is booted up.
This DHCP client accepts few vendor specific options which in turn allow for at least the following:

* What IP address the device should use to:
    * Contact a server using tftp, http etc.
    * Potentially updgrade(/downgrade?) software to the requested release from the server.
    * Download a configuration file from the server and try committing it.

As usual, the device is identified based on its mac address and this can be used to determine which config file to download.

## Sample ISC DHCP config ##

Below sample is largely based on [docs](https://www.juniper.net/documentation/us/en/software/junos/junos-install-upgrade/topics/topic-map/zero-touch-provision.html) available from Juniper.

```
DHCP-SERVER:/var/tftpboot# cat /etc/dhcp/dhcpd.conf
option space NEW_OP;
option NEW_OP.image-file-name code 0 = text;
option NEW_OP.config-file-name code 1 = text;
option NEW_OP.image-file-type code 2 = text; 
option NEW_OP.transfer-mode code 3 = text;
option NEW_OP.alt-image-file-name code 4= text;
option NEW_OP.http-port code 5= text;
option NEW_OP-encapsulation code 43 = encapsulate NEW_OP;
option NEW_OP.proxyv4-info code 8 = text;


subnet 10.100.123.0 netmask 255.255.255.0 {

    range 10.100.123.10 10.100.123.50;

}

 host SPINE-1-A {                            
        hardware ethernet 0c:04:5e:ca:00:00;     
        fixed-address 10.100.123.123;            
                                                 
        option tftp-server-name "10.100.123.254";
        #option NEW_OP.ftp-timeout ...val...;
        #option host-name "ztp-test";                                                         
        #option log-servers 10.100.31.72;                                                     
        #option ntp-servers 10.100.31.73;                                                     
        #option NEW_OP.image-file-name "/junos/sw/jinstall-host-qfx-5-21.4R3-S3.4-signed.tgz";
        option NEW_OP.transfer-mode "tftp";            
        #option NEW_OP.http-port code 5= 80;           
        option NEW_OP.config-file-name "/SPINE-1-A.cfg";
    }



 host ztp-test { 
        hardware ethernet 0c:fc:be:62:00:00; 
        fixed-address 10.100.123.123; 
    
        option tftp-server-name "10.100.123.254"; 
        #option NEW_OP.ftp-timeout “val”;
        #option host-name "ztp-test"; 
        #option log-servers 10.100.31.72; 
        #option ntp-servers 10.100.31.73; 
        #option NEW_OP.image-file-name "/junos/sw/jinstall-host-qfx-5-21.4R3-S3.4-signed.tgz"; 
        option NEW_OP.transfer-mode "tftp"; 
        #option NEW_OP.http-port code 5= 80;
        option NEW_OP.config-file-name "/LEAF-1-A.cfg"; 
    } 
DHCP-SERVER:/var/tftpboot# 

```

I connected the sampling switches via their fxp0 to a 'Out Of Band' management switch that also connected to a Alpine Linux at 10.100.123.254 that had tftp-hpa installed.

```
apk add tftp-hpa
rc-update add in.tftpd

#By default the tftproot resides in /var/tftpboot
DHCP-SERVER:/var/tftpboot# ls -latr /var/tftpboot/
total 20
drwxr-xr-x 12 root root 4096 Nov 24 14:57 ..
-rw-r--r--  1 root root 6396 Nov 24 15:02 LEAF-1-A.cfg
-rw-r--r--  1 root root 4003 Nov 24 15:34 SPINE-1-A.cfg
drwxr-xr-x  2 root root 4096 Nov 24 15:34 .
DHCP-SERVER:/var/tftpboot# 

```

Now when the switches boot they eventually download the configs and apparently also successfully commit those.


