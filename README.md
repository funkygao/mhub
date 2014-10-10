mhub
====

Message hub, a real-time MQTT broker that supports cluster.

Thanks to https://github.com/jeffallen/mqtt

### Why MQTT

* pub/sub model
* bidirectionnal
* complement to enterprise messaging systems so that a wealth of data outside the enterprise could be safely and easily brought inside the enterprise
* lightweight 
* network(especially mobile) is not 100% reliable
* mobile battery friendly
* max payload 256MB, just bytes array without format
* QoS on a per-message basis
* TLS & user/pass authentication/authorization
* rich client API with all of 5 protocol methods
  - connect
    - ping(internally)
  - disconnect
  - publish
  - subscribe
  - unsubscribe

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


### Network I/O

                 etcd
                  |
                  |     +- peer
                mhub ---|- peer
                  |     +- peer
                  |      
          +----------------+
          |       |        |
        client  client  client

### Clustering

* no SPOF
* a client PUBLISH to any node in a cluster will be pushed to all the topic subscribers in the cluster
    - PUBLISH is replicated across all broker nodes(peers) in the cluster
* broker nodes uses etcd for service auto discovery
* new broker node automatically joins a cluster

### Pitfalls

* client id
  - clients need to have an identifier that is unique for all clients connecting to the broker
* message id

### TODO
*   cluster of brokers, scales with the number of MQTT clients
*   ForceDisconnect after heartbeat idle too long
*   security
*   connect/io timeout of client/peers
*   more edge cases testing
*   retain, will
    - with retain flag true, the published message is held onto by the broker, so when the late arrivers connect to the broker or clients create a new subscription they get all the relevant retained messages.
*   persistency
