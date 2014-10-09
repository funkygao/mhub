// Package mqtt implements MQTT clients and servers.
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
              reader ----------- writer

      
*/
package broker
