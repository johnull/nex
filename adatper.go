package nex

import (
	"encoding/json"
	"net/http"
	"reflect"
)

type HandlerAdapter interface {
	Invoke(http.ResponseWriter, *http.Request)
}

type genericAdapter struct {
	method reflect.Value
	numIn  int
	types  []reflect.Type
}

// Accept zero parameter adapter
type noneParameterAdapter struct {
	method reflect.Value
}

// Accept only one parameter adapter
type simpleUnmarshalAdapter struct {
	argType reflect.Type
	method  reflect.Value
}

func makeGenericAdapter(method reflect.Value) *genericAdapter {
	var noSupportExists = false
	t := method.Type()
	numIn := t.NumIn()

	a := &genericAdapter{
		method: method,
		numIn:  numIn,
		types:  make([]reflect.Type, numIn),
	}

	for i := 0; i < numIn; i++ {
		in := t.In(i)
		if !isSupportType(in) {
			if noSupportExists {
				panic("function should accept only one customize type")
			}

			if in.Kind() != reflect.Ptr {
				panic("customize type should be a pointer(" + in.PkgPath() + "." + in.Name() + ")")
			}
			noSupportExists = true
		}
		a.types[i] = in
	}

	return a
}

func (a *genericAdapter) Invoke(w http.ResponseWriter, r *http.Request) {
	values := make([]reflect.Value, a.numIn)
	for i := 0; i < a.numIn; i++ {
		v, ok := supportTypes[a.types[i]]
		if ok {
			values[i] = v(r)
		} else {
			d := reflect.New(a.types[i].Elem()).Interface()
			err := json.NewDecoder(r.Body).Decode(d)
			if err != nil {
				fail(w, err)
				return
			}
			values[i] = reflect.ValueOf(d)
		}
	}

	ret := a.method.Call(values)
	if err := ret[1].Interface(); err != nil {
		fail(w, err.(error))
		return
	}

	succ(w, ret[0].Interface())
}

func (a *noneParameterAdapter) Invoke(w http.ResponseWriter, r *http.Request) {
	ret := a.method.Call([]reflect.Value{})
	if err := ret[1].Interface(); err != nil {
		fail(w, err.(error))
		return
	}

	succ(w, ret[0].Interface())
}

func (a *simpleUnmarshalAdapter) Invoke(w http.ResponseWriter, r *http.Request) {
	data := reflect.New(a.argType.Elem()).Interface()
	err := json.NewDecoder(r.Body).Decode(data)
	if err != nil {
		fail(w, err)
		return
	}

	ret := a.method.Call([]reflect.Value{reflect.ValueOf(data)})
	if err := ret[1].Interface(); err != nil {
		fail(w, err.(error))
		return
	}

	succ(w, ret[0].Interface())
}