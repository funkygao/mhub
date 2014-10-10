mhub
====

Message hub, a real-time MQTT broker that supports cluster.

### Arch

            MQTT-client   MQTT-client    MQTT-client 
              |               |                |
              +--------------------------------+
                              |
                      publish | subscribe
                              |
                              |          mhub cluster
               +------------------------------------+
               |                                    |
               |             PUBLISH                |
               |           replication              |
               | mhub node -------------- mhub node |
               |          \              /          |
               |           \           /            |
               |            \        /              |
               |            mhub node               |
               |                                    |
               +------------------------------------+
                              |
                              | service discovery
                              |
                          etcd cluster



### Clustering

* a client connecting to any node in a cluster can see all topics in the cluster.
* broker nodes uses etcd for service auto discovery
* new broker node automatically joins a cluster
* PUBLISH is replicated across all broker nodes
* no SPOF

### TODO
*   cluster of brokers, scales with the number of MQTT clients
*   ForceDisconnect after heartbeat idle too long
*   errUseClosedConn
