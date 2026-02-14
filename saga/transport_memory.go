package saga

type InMemoryTransport struct{}

func NewInMemoryTransport() *InMemoryTransport {
	return &InMemoryTransport{}
}

func (t *InMemoryTransport) Publish(event Event) error {
	return nil
}

func (t *InMemoryTransport) Close() error {
	return nil
}
