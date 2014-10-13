// Package broker implements MQTT client and server/broker.
/*
                                            +- worker1 -+
                 Server -- subscriptions ---|- worker2 -|-- chan post
                                            +- workerN -+
                   |                             |
                   |           +-----------------+
                   |           |
       +-------------------------------+
       |               |               |
   incomingConn    incomingConn    incomingConn
                       |
               +-------------------+
               |                   |
               |     chan job      |
            inbound ----------- outbound
               ^                   V
               |                   |
               +-------------------+
                       |
                     client
*/
package broker
