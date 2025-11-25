package reset

type PoolNaive[T Resetter] struct {
	Queue []T
	New   func() any
}

func (p *PoolNaive[T]) Put(item T) {
	item.Reset()
	p.Queue = append(p.Queue, item)

}

func (p *PoolNaive[T]) Get() T {
	if len(p.Queue) > 0 {
		result := p.Queue[0]
		p.Queue = p.Queue[:1]
		return result
	}
	return p.New().(T)
}

func NewPoolNaive[T Resetter](newF func() T) *PoolNaive[T] {
	return &PoolNaive[T]{
		Queue: make([]T, 0),
		New: func() any {
			return newF()
		},
	}
}
