package a

func f() {
	var x error
	Wrap(x) // want "xx"
	var s S
	s.Wrap(nil) // want "xx"
}

func Wrap(x any) {}

type S int

func (s S) Wrap(x any) {}
