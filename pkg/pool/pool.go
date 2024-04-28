package pool

type Pool interface {
	Get() (interface{}, error)

	Put(interface{}) error

	Close(interface{}) error

	Release()

	Len() int
}
