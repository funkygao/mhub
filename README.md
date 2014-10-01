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
               |              batch                 |
               |    broker ------------ broker      |
               |                                    |
               +------------------------------------+


### Topic

System/SubSystem0/Device0/Param0

### Broker

*   centralized broker
*   clustered broker
*   distributed broker

### TODO
*   rename to mhub
*   cluster of brokers, scales with the number of MQTT clients
