package mqtt

import (
	"sync"
)

/* Glabal status */
var G_clients map[string]*Client = make(map[string]*Client)
var G_clients_lock *sync.Mutex = new(sync.Mutex)

// Map topic => sub
// sub is implemented as map, key is client_id, value is qos
var G_subs map[string]map[string]uint8 = make(map[string]map[string]uint8)
var G_subs_lock *sync.Mutex = new(sync.Mutex)
var G_redis_client *RedisClient = StartRedisClient()

var g_redis_lock *sync.Mutex = new(sync.Mutex)

var NextClientMessageId map[string]uint16 = make(map[string]uint16)
var g_next_client_id_lock *sync.Mutex = new(sync.Mutex)

var G_topicss map[string]*Topic = make(map[string]*Topic)
var G_topics_lockk *sync.Mutex = new(sync.Mutex)
