package mqtt

import (
	"fmt"
	log "github.com/cihub/seelog"
)

// This function should be called upon starting up. It will
// try to recover global status(G_subs, etc) from Redis.
func RecoverFromRedis() {
	// Reconstruct the subscription map from redis records
	client_id_keys := G_redis_client.GetSubsClients()
	log.Info("Recovering subscription info from Redis")
	for _, client_id_key := range client_id_keys {
		sub_map := make(map[string]uint8)
		G_redis_client.Fetch(client_id_key, &sub_map)
		var client_id string
		fmt.Sscanf(client_id_key, "gossipd.client-subs.%s", &client_id)

		for topic, qos := range sub_map {
			// lock won't be needed since this is at the startup phase
			subs := G_subs[topic]
			if subs == nil {
				log.Debug("current subscription is the first client to topic:", topic)
				subs = make(map[string]uint8)
				G_subs[topic] = subs
			}
			subs[client_id] = qos
		}
		log.Debugf("client(%s) subscription info recovered", client_id)
	}
}
