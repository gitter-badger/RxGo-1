package observable

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/jochasinga/grx/bases"
	"github.com/jochasinga/grx/emittable"
	"github.com/jochasinga/grx/handlers"
	"github.com/jochasinga/grx/iterable"
	"github.com/jochasinga/grx/observer"

	"github.com/stretchr/testify/assert"
)

func TestObservableImplementStream(t *testing.T) {
	assert.Implements(t, (*bases.Stream)(nil), DefaultObservable)
}

func TestObservableImplementIterator(t *testing.T) {
	assert.Implements(t, (*bases.Iterator)(nil), DefaultObservable)
}

func TestCreateObservableWithConstructor(t *testing.T) {
	source := New()
	assert.IsType(t, (*Observable)(nil), source)
}

func TestCreateOperator(t *testing.T) {
	source := Create(func(ob *observer.Observer) {
		ob.OnNext("Hello")
	})

	empty := ""

	eventHandler := handlers.NextFunc(func(it bases.Item) {
		if text, ok := it.(string); ok {
			empty += text
		} else {
			panic("Item is not a string")
		}
	})

	_, _ = source.Subscribe(eventHandler)
	<-time.After(100 * time.Millisecond)
	assert.Equal(t, "Hello", empty)

	source = Create(func(ob *observer.Observer) {
		ob.OnError(errors.New("OMG this is an error"))
	})

	errText := ""
	errHandler := handlers.ErrFunc(func(err error) {
		errText += err.Error()
	})
	_, _ = source.Subscribe(errHandler)
	<-time.After(100 * time.Millisecond)
	assert.Equal(t, "OMG this is an error", errText)
}

func TestEmptyOperator(t *testing.T) {
	msg := "Sumpin's"
	source := Empty()

	watcher := &observer.Observer{
		NextHandler: handlers.NextFunc(func(i bases.Item) {
			panic("NextHandler shouldn't be called")
		}),
		DoneHandler: handlers.DoneFunc(func() {
			msg += " brewin'"
		}),
	}
	_, err := source.Subscribe(watcher)
	assert.Nil(t, err)

	<-time.After(100 * time.Millisecond)
	assert.Equal(t, "Sumpin's brewin'", msg)
}

func TestJustOperator(t *testing.T) {
	assert := assert.New(t)
	url := "http://api.com/api/v1.0/user"
	source := Just(url)

	assert.IsType((*Observable)(nil), source)

	urlWithQueryString := ""
	queryString := "?id=999"
	expected := url + queryString

	watcher := &observer.Observer{
		NextHandler: handlers.NextFunc(func(it bases.Item) {
			if url, ok := it.(string); ok {
				urlWithQueryString += url
			}
		}),
		DoneHandler: handlers.DoneFunc(func() {
			urlWithQueryString += queryString
		}),
	}

	sub, err := source.Subscribe(watcher)
	assert.Nil(err)
	assert.NotNil(sub)
	<-time.After(10 * time.Millisecond)
	assert.Equal(expected, urlWithQueryString)

	source = Just('R', 'x', 'G', 'o')
	e, err := source.Next()
	assert.IsType((*Observable)(nil), source)
	assert.Nil(err)
	assert.IsType((*emittable.Emittable)(nil), e)
	assert.Implements((*bases.Emitter)(nil), e)
}

func TestFromOperator(t *testing.T) {
	assert := assert.New(t)

	iterableUrls := iterable.From([]interface{}{
		"http://api.com/api/v1.0/user",
		"https://dropbox.com/api/v2.1/get/storage",
		"http://googleapi.com/map",
	})

	responses := []string{}

	request := func(url string) string {
		randomNum := rand.Intn(100)
		time.Sleep((200 - time.Duration(randomNum)) * time.Millisecond)
		return fmt.Sprintf("{\"url\":%q}", url)
	}

	urlStream := From(iterableUrls)
	urlObserver := &observer.Observer{
		NextHandler: handlers.NextFunc(func(it bases.Item) {
			if url, ok := it.(string); ok {
				res := request(url)
				responses = append(responses, res)
			} else {
				assert.Fail("Item is not a string as expected")
			}
		}),
		DoneHandler: handlers.DoneFunc(func() {
			responses = append(responses, "END")
		}),
	}

	sub, err := urlStream.Subscribe(urlObserver)
	assert.NotNil(sub)
	assert.Nil(err)

	<-time.After(100 * time.Millisecond)
	expectedStrings := []string{
		"{\"url\":\"http://api.com/api/v1.0/user\"}",
		"{\"url\":\"https://dropbox.com/api/v2.1/get/storage\"}",
		"{\"url\":\"http://googleapi.com/map\"}",
		"END",
	}
	assert.Exactly(expectedStrings, responses)

	iterableNums := iterable.From([]interface{}{1, 2, 3, 4, 5, 6})
	numCopy := []int{}

	numStream := From(iterableNums)
	numObserver := &observer.Observer{
		NextHandler: handlers.NextFunc(func(it bases.Item) {
			if num, ok := it.(int); ok {
				numCopy = append(numCopy, num+1)
			} else {
				assert.Fail("Item is not an integer as expected")
			}
		}),
		DoneHandler: handlers.DoneFunc(func() {
			numCopy = append(numCopy, 0)
		}),
	}

	sub, err = numStream.Subscribe(numObserver)
	assert.NotNil(sub)
	assert.Nil(err)

	<-time.After(100 * time.Millisecond)
	expectedNums := []int{2, 3, 4, 5, 6, 7, 0}
	assert.Exactly(expectedNums, numCopy)
}

func TestStartOperator(t *testing.T) {
	assert := assert.New(t)
	d1 := func() bases.Emitter {
		return emittable.From(333)
	}
	d2 := func() bases.Emitter {
		return emittable.From(666)
	}
	d3 := func() bases.Emitter {
		return emittable.From(999)
	}

	source := Start(d1, d2, d3)
	e, err := source.Next()

	assert.IsType((*Observable)(nil), source)
	assert.Nil(err)
	assert.IsType((*emittable.Emittable)(nil), e)
	assert.Implements((*bases.Emitter)(nil), e)

	nums := []int{}
	watcher := &observer.Observer{
		NextHandler: handlers.NextFunc(func(it bases.Item) {
			if num, ok := it.(int); ok {
				nums = append(nums, num+111)
			} else {
				assert.Fail("Item is not an integer as expected")
			}
		}),
		DoneHandler: handlers.DoneFunc(func() {
			nums = append(nums, 0)
		}),
	}

	_, _ = source.Subscribe(watcher)
	<-time.After(100 * time.Millisecond)
	expected := []int{444, 777, 1110, 0}
	assert.Exactly(expected, nums)
}

func TestStartMethodWithFakeExternalCalls(t *testing.T) {
	fakeHttpResponses := []*http.Response{}

	// NOTE: HTTP Response errors such as status 500 does not return an error
	fakeHttpErrors := []error{}

	// Fake directives that returns an Event containing an HTTP response.
	d1 := func() bases.Emitter {
		res := &http.Response{
			Status:     "404 NOT FOUND",
			StatusCode: 404,
			Proto:      "HTTP/1.0",
			ProtoMajor: 1,
		}

		// Simulating an I/O block
		time.Sleep(20 * time.Millisecond)
		return emittable.From(res)
	}

	d2 := func() bases.Emitter {
		res := &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Proto:      "HTTP/1.0",
			ProtoMajor: 1,
		}
		time.Sleep(10 * time.Millisecond)
		return emittable.From(res)
	}

	d3 := func() bases.Emitter {
		res := &http.Response{
			Status:     "500 SERVER ERROR",
			StatusCode: 500,
			Proto:      "HTTP/1.0",
			ProtoMajor: 1,
		}
		time.Sleep(30 * time.Millisecond)
		return emittable.From(res)
	}

	d4 := func() bases.Emitter {
		err := errors.New("Some kind of error")
		time.Sleep(50 * time.Millisecond)
		return emittable.From(err)
	}

	watcher := &observer.Observer{
		NextHandler: handlers.NextFunc(func(it bases.Item) {
			if res, ok := it.(*http.Response); ok {
				fakeHttpResponses = append(fakeHttpResponses, res)
			}
		}),
		ErrHandler: handlers.ErrFunc(func(err error) {
			fakeHttpErrors = append(fakeHttpErrors, err)
		}),
		DoneHandler: handlers.DoneFunc(func() {
			fakeHttpResponses = append(fakeHttpResponses, &http.Response{
				Status:     "999 End",
				StatusCode: 999,
			})
		}),
	}

	source := Start(d1, d2, d3, d4)
	_, err := source.Subscribe(watcher)

	assert := assert.New(t)
	assert.Nil(err)

	<-time.After(100 * time.Millisecond)

	assert.IsType((*Observable)(nil), source)
	assert.Equal(4, len(fakeHttpResponses))
	assert.Equal(1, len(fakeHttpErrors))
	assert.Equal(200, fakeHttpResponses[0].StatusCode)
	assert.Equal(404, fakeHttpResponses[1].StatusCode)
	assert.Equal(500, fakeHttpResponses[2].StatusCode)
	assert.Equal(999, fakeHttpResponses[3].StatusCode)
	assert.Equal("Some kind of error", fakeHttpErrors[0])
}

func TestIntervalOperator(t *testing.T) {
	assert := assert.New(t)
	numch := make(chan int, 1)
	source := Interval(1 * time.Millisecond)
	assert.IsType((*Observable)(nil), source)

	go func() {
		_, _ = source.Subscribe(&observer.Observer{
			NextHandler: handlers.NextFunc(func(it bases.Item) {
				if num, ok := it.(int); ok {
					numch <- num
				}
			}),
		})
	}()

	i := 0

	select {
	case <-time.After(1 * time.Millisecond):
		if i >= 10 {
			return
		}
		i++
	case num := <-numch:
		assert.Equal(i, num)
	}
	/*
		for i := 0; i <= 10; i++ {
			<-time.After(1 * time.Millisecond)
			assert.Equal(i, <-numch)
		}
	*/
}

func TestRangeOperator(t *testing.T) {
	assert := assert.New(t)
	nums := []int{}
	watcher := &observer.Observer{
		NextHandler: handlers.NextFunc(func(it bases.Item) {
			if num, ok := it.(int); ok {
				nums = append(nums, num)
			} else {
				assert.Fail("Item is not an integer as expected")
			}
		}),
	}
	source := Range(1, 10)
	assert.IsType((*Observable)(nil), source)

	_, err := source.Subscribe(watcher)
	assert.Nil(err)

	<-time.After(100 * time.Millisecond)
	expected := []int{1, 2, 3, 4, 5, 6, 7, 8, 9}
	assert.Exactly(expected, nums)
}

func TestSubscriptionIsNonBlocking(t *testing.T) {
	var (
		s1 = Just("Hello", "world", "foo", 'a', 1.2, -3111.02, []rune{}, struct{}{})
		s2 = From(iterable.From([]interface{}{1, 2, "Hi", 'd', 2.10, -54, []byte{}}))
		s3 = Range(1, 100)
		s4 = Interval(1 * time.Second)
		s5 = Empty()
	)

	sources := []*Observable{s1, s2, s3, s4, s5}

	watcher := &observer.Observer{
		NextHandler: handlers.NextFunc(func(it bases.Item) {
			time.Sleep(1 * time.Second)
			t.Log(it)
			return
		}),
		DoneHandler: handlers.DoneFunc(func() {
			t.Log("DONE")
		}),
	}

	first := time.Now()

	for _, source := range sources {
		_, err := source.Subscribe(watcher)
		assert.Nil(t, err)
	}

	elapsed := time.Since(first)

	comp := assert.Comparison(func() bool {
		return elapsed < 1*time.Second
	})
	assert.Condition(t, comp)
}

func TestObservableDoneChannel(t *testing.T) {
	assert := assert.New(t)
	o := Range(1, 10)
	_, err := o.Subscribe(&observer.Observer{
		NextHandler: handlers.NextFunc(func(it bases.Item) {
			t.Logf("Test value: %v\n", it)
		}),
		DoneHandler: handlers.DoneFunc(func() {
			o.notifier.Done()
		}),
	})

	assert.Nil(err)

	<-time.After(100 * time.Millisecond)
	assert.Equal(struct{}{}, <-o.notifier.IsDone)
}

func TestObservableGetDisposedViaSubscription(t *testing.T) {
	nums := []int{}
	source := Interval(100 * time.Millisecond)
	sub, err := source.Subscribe(&observer.Observer{
		NextHandler: handlers.NextFunc(func(it bases.Item) {
			if num, ok := it.(int); ok {
				nums = append(nums, num)
			} else {
				assert.Fail(t, "Item is not an integer as expected")
			}
		}),
	})

	assert.Nil(t, err)
	assert.NotNil(t, sub)

	<-time.After(300 * time.Millisecond)
	sub.Dispose()

	comp := assert.Comparison(func() bool {
		return len(nums) <= 4
	})
	assert.Condition(t, comp)
}
