# go-DDDNS
Go implementation of the Routing ID based Distributed Dynamic Domain Name Service

## Introduction

There is no such standard as DDDNS and this is not a DDNS protocol. I drafted this experiment myself to test if it is possible to create a clusters that "floats" on the floating IP assigned by the ISP or in a network environment that can only allow plug-and-play hosting.

The requirement of this project is as follows.

1. The software cannot get IP address from its NIC, or using any kind of commands that request OS to provide the IP address information (e.g. No ip a or ifconfig)
2. No external dependencies other than the connected nodes (e.g. no UPnP to ask router what is the current node IP, no online IP checking API)
3. No platform dependencies (aka it should not be only working on Linux)

All of the ip address information has to be come from its connected nodes and the packets they are sending each others, manual input is acceptable only during the initial setting up and the software itself should handle all the ip changes and map itself correctly in the cluster / node mesh.

If this project is proofed to be working and secure, this will be added to the ArozOS project as the fundamental section of its Clustering System.



## License

All right reserved 

(Will switch to open source license later after this is completed. For now, please don't do anything with it, **especially don't use it in production**, however comments and ideas are welcomed via **issues**)

