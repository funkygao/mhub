package mqtt

type Session struct {
	Protocol   string
	Version    uint8
	Keepalive  uint16
	WillRetain bool
	WillQos    uint8
	WillFlag   bool
}
