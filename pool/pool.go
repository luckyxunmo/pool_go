package pool

import (
	"sync"
	"io"
	"github.com/CodisLabs/codis/pkg/utils/errors"
	"log"
)

type Pool struct{
	m sync.Mutex
	resources chan io.Closer
	factory func()(io.Closer,error)
	closed bool
}

var ErrPoolClosed = errors.New("Pool has been closed")

func New(fn func()(io.Closer,error),size uint)(*Pool, error) {
	if size <= 0{
		return nil,errors.New("size value too small")
	}
	return &Pool{
		factory:fn,
		resources:make(chan io.Closer,size),
	},nil
}
// Acquire retrieves a resource from the pool
func (p *Pool)Acquire()(io.Closer,error)  {
	select{
	 case r,ok := <-p.resources:
	 	log.Println("Acquire","shared Resource")
	 	if !ok{
	 		return nil,ErrPoolClosed
		}
		return r,nil
	default:
		log.Println("Acquire:","New Resource")
		return p.factory()
	}
}
// Release places a new resource onto the pool.
func (p *Pool)Release(r io.Closer)  {
	p.m.Lock()
	defer p.m.Unlock()
	if p.closed{
		r.Close()
		return
	}
	select{
	 case p.resources <- r:
	 	log.Println("Release:","In queue")
	default:
		log.Println("Release","closing")
		r.Close()

	}
}

func (p *Pool)Close()  {
	p.m.Lock()
	defer p.m.Unlock()
	if p.closed{
		return
	}
	p.closed = true
	// Close the channel before we drain the channel of its
	//resources. If we don't do this, we will have a deadlock.
	close(p.resources)
	//close the resource
	for r := range p.resources{
		r.Close()
	}
}