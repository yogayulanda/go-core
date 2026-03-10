package messaging

type Message struct {
	Topic   string
	Key     []byte
	Payload []byte
	Headers map[string]string
}
