package a

import "errors"

func g() {
	err := doSomething()
	if err != nil {
		panic("panic")
	}
	Wrap(err) // want "xx"
}

func h() (err error) {
	Wrap(err) // want "xx"
	err = errors.New("hoge")
	Wrap(err) // ok
	return err
}

func i() {
	Wrap(nil) // want "xx"
}

func f() {
	var s []int = nil
	Wrap(s) // want "xx"
	ss := s[:][:]
	Wrap(ss) // want "xx"
}

func f3() (a []int) {
	Wrap(a) // want "xx"
	return
}

func f2(x any) {
	switch x.(type) {
	case nil:
		Wrap(x) // want "xx"
	case int:
		Wrap(x) // ok
	}
}

func doSomething() error {
	return nil
}

func Wrap(x any) {}

type S int

func (s S) Wrap(x any) {}
