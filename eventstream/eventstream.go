package eventstream

import (
	"fmt"

	"github.com/jochasinga/grx/bases"
	"github.com/jochasinga/grx/errors"
)

type EventStream chan bases.Emitter

// Next returns the next Event on the EventStream
func (evs EventStream) Next() (bases.Emitter, error) {
	if emitter, ok := <-evs; ok {
		return emitter, nil
	}
	return nil, NewError(errors.EndOfIteratorError)
}

// New creates a new EventStream from one or more Event
func New(emitters ...bases.Emitter) EventStream {
	es := make(EventStream, len(emitters))
	if len(emitters) > 0 {
		go func() {
			for _, emitter := range emitters {
				es <- emitter
			}
			close(es)
		}()
	}
	return es
}

// From creates a new EventStream from an Iterator
func From(iter bases.Iterator) EventStream {
	es := make(EventStream)
	go func() {
		for {
			emitter, err := iter.Next()
			fmt.Println(emitter, err)
			if err != nil {
				break
			}
			es <- emitter
		}
		close(es)
	}()
	return es
}
