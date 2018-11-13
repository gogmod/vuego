// +build js,wasm

package vuego

import (
	"fmt"
	"reflect"
	"syscall/js"

	"github.com/segmentio/ksuid"
)

var (
	obj = js.Global().Get("Object")
	arr = js.Global().Get("Array")
)

type Vue struct {
	js.Value
	El   string
	Data interface{}

	Methods *Methods
	Filters *Filters
	Watch   *Watch

	Created func(interface{}, []js.Value)
	Mounted func(interface{}, []js.Value)
}

func New(v *Vue) *Vue {
	arg := obj.New()
	arg.Set("el", v.El)
	arg.Set("data", ToJson(v.Data))

	if v.Watch != nil {
		arg.Set("watch", js.ValueOf(*v.Watch))
	}

	if v.Created != nil {
		arg.Set("created", js.NewCallback(func(args []js.Value) {
			v.Created(v.Data, args)
		}))
	}

	if v.Filters != nil {
		filters := obj.New()
		for name, fn := range *v.Filters {
			id := "VUEGO_FILTER_" + ksuid.New().String()
			output := id + "OUTPUT"
			cb := js.NewCallback(func(args []js.Value) {
				js.Global().Set(output, js.ValueOf(fn(args[0])))
			})
			js.Global().Set(id, cb)
			wrapper := js.Global().Call("eval", fmt.Sprintf("(function(v){ %s(v); return global.%s; })", id, output))
			filters.Set(name, wrapper)
		}
		arg.Set("filters", filters)
	}

	if v.Methods != nil {
		methods := obj.New()
		for name, fn := range *v.Methods {
			cb := js.NewCallback(func(args []js.Value) {
				fn(v.Data)
			})
			methods.Set(name, js.ValueOf(cb))
		}
		arg.Set("methods", methods)
	}
	v.Value = js.Global().Get("Vue").New(arg)
	return v
}

func ValueOf(value reflect.Value) js.Value {
	switch value.Kind() {
	case reflect.Slice:
		a := arr.New(value.Len())
		for i := 0; i < value.Len(); i++ {
			a.SetIndex(i, ToJson(value.Index(i).Interface()))
		}
		return a
	default:
		return js.ValueOf(value.Interface())
	}
}

func ToJson(content interface{}) js.Value {
	value := obj.New()
	v := reflect.Indirect(reflect.ValueOf(content))
	r := v.Type()

	if r.Name() == "string" {
		return js.ValueOf(v.Interface())
	}

	for i := 1; i < r.NumField(); i++ {
		name := r.Field(i).Tag.Get("json")
		if name != "" {
			value.Set(name, ValueOf(v.Field(i)))
		}
	}

	v.Field(0).Set(reflect.ValueOf(value))

	return value
}
