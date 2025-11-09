package reset

import (
	"sync"
)

type Resetter interface {
	Reset()
}

type Pool[T Resetter] struct {
	sync.Pool
}

func (p *Pool[T]) Get() T {
	return p.Pool.Get().(T)
}

func (p *Pool[T]) Put(x T) {
	p.Pool.Put(x)
}

func NewPool[T Resetter](newF func() T) *Pool[T] {
	return &Pool[T]{
		Pool: sync.Pool{
			New: func() interface{} {
				return newF()
			},
		},
	}
}
