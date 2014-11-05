mhub
====

Message hub, a real-time MQTT v3.1 broker that supports cluster.
                                                       
               _           _     
              | |         | |    
     _ __ ___ | |__  _   _| |__  
    | '_ ` _ \| '_ \| | | | '_ \ 
    | | | | | | | | | |_| | |_) |
    |_| |_| |_|_| |_|\__,_|_.__/ 
                                 

Thanks to https://github.com/jeffallen/mqtt

### Why MQTT

* a lightweight bidirectionnal protocol of pub/sub/push model for constrained environment and devices
* support for loss of contact between c/s(last will)
* mobile battery friendly
* max payload 256MB, just bytes array without format
* topic can be 64KB long
* QoS on a per-message basis
* TLS & user/pass authentication/authorization
* rich client API with all of 5 protocol methods
  - connect, publish, (un)subscribe, disconnect
* availability of client libraries

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


### Features

*   QoS1 fully implemented
*   client rate limit
*   authentication and authorization RBAC
*   memory/redis based flying message store
*   presense service(not implemented)
*   recyclable memory pool to reduce GC
*   full stats report with REST interface
*   full benchmark tests
*   clustering
    - no SPOF
    - a client PUBLISH to any node in a cluster will be pushed to all the topic subscribers in the cluster
       - PUBLISH is replicated across all broker nodes(peers) in the cluster
    - broker nodes uses etcd for service auto discovery
    - new broker node automatically joins a cluster

### Capacity Plan

* throughput
  - 50K message/sec per broker
* max concurrenty conn
  - about 40KB per connection(100K conn is 4GB)
  - vitess typically run 5-20K connections and rarely exceed 1GB
  - 600K concurrent tcp conns consumes 16GB with each 28KB

### Pitfalls

* client id
  - clients need to have an identifier that is unique for all clients connecting to the broker
  - can be token, but max len is 23
* message id
  - uint16
  - PUBLISH messages with QoS1/2 require a message id as part of the packet
  - are handled on a per client and per direction basis

### TODO

*   peers broadcast Subscribe problem is new broker don't know subscribed endpoints
*   why job chan got full under loadtest
*   cluster of brokers, scales with the number of MQTT clients
*   GC
    - gc and scavenger
    - the more objects there are, the more expensive garbage collection is
    - the more pointers we need to chase, the more expensive gc is
    - recycling mem buffer to avoid trigger GC
    - GOGC mgc0.c
    - GOTRACEBACK
*   more edge cases testing
*   circuit breaker for peer
*   user lock?
*   retain, last will, clean session
    - retain is 'last known good value'
    - with retain flag true, the published message is held onto by the broker, so when the late arrivers connect to the broker or clients create a new subscription they get all the relevant retained messages.
*   persistency of messages
*   c(pub)->s(pub to subs), msg id may conflict

### C1000K

https://github.com/xiaojiaqi/C1000kPracticeGuide
