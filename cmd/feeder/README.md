feeder
======

Reside on each web server and consumes php's push cmds through rsyslogd, then
publish to mhub through persistent connections.
For a given user's msg, must be ordered(flying message lock/scheduling).

### Arch

                    client -----------------+
                      |                     |
                      | http                |
                      |                     |
              +---------------------+       |
              | php-fpm             |       |
              |    |                |       |
              |    | 30us           |       |
              |    | json lines     |       ^ push
              |    |                |       |
              |  feeder             |       |
              |    |                |       |
              +----|----------------+       |
                   |                        |
                   V 2ms                    |
                   |                        |
                  mhub ---------------------+
                   

