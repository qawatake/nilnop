package a

import "errors"

func g1() {
	err := doSomething()
	if err != nil {
		panic(err)
	}
	s.Wrap(err) // want "xx"
}

func g2() (err error) {
	s.Wrap(err) // want "xx"
	err = errors.New("hoge")
	s.Wrap(err) // ok
	return err
}

func g3() {
	s.Wrap(nil) // want "xx"
}

func g4(x any) {
	switch x.(type) {
	case nil:
		s.Wrap(x) // want "xx"
	case int:
		s.Wrap(x) // ok
	}
}

type S int

var s S

func (s S) Wrap(x any) {}
