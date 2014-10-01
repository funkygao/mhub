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
                              | MQTT
                              |               cluster
               +------------------------------------+
               |                                    |
               |              batch                 |
               |    broker ------------ broker      |
               |                                    |
               +------------------------------------+


### TODO
*   rename to mhub
*   cluster of brokers
