package functions

type Invoker interface {
	Invoke(name string, payload []byte) error
}
