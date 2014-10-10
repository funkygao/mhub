gomqtt
======

a MQTT broker in golang.

The MQTT protocol relies on a message broker according to the hub and spoke model 
of Message Oriented Middleware (MOM).


### Arch

            client          client          client
              |                |               |
              +--------------------------------+
                              |
                      publish | subscribe
                              |
                              | msg(cmd/data)
                              |
                              | MQTT
                              |               cluster
               +------------------------------------+
               |                                    |
               |           replication              |
               |    broker ------------ broker      |
               |                                    |
               +------------------------------------+


### Topic

System/SubSystem0/Device0/Param0

### Broker

*   centralized broker
*   clustered broker
*   distributed broker

### Clustering

Clustering connects multiple machines together to form a single logical broker.

A client connecting to any node in a cluster can see all topics in the cluster.

All data/state required for the operation of  MQTT broker is replicated across all nodes, for scaling and reliability
with full ACID properties.
An exception to this are message topics, which by default reside on the node that created them, though they are visible 
and reachable from all nodes.

MQTT clustering does not tolerate network partitions well, so it should not be used over a WAN.

New node automatically joins a cluster.

### TODO
*   cluster of brokers, scales with the number of MQTT clients
*   ForceDisconnect
