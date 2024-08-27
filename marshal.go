package bencode

import (
	"errors"
	"io"
	"reflect"
	"string"

	"golang.org/x/tools/go/analysis/passes/unmarshal"
)

// reflect: type interface{}; value {e.typ, e.word, flag}
func unmarshalList(p reflect.Value, list []*BObject) error {
	// check if the p is a pointer and ensure the value is addressable
	// p is expected to be a pointer to a slice, not slice itself
	if p.Kind() != reflect.Ptr || p.Elem().Kind() != reflect.Slice {
		return errors.New("dest must be pointet to slice")
	}
	v := p.Elem() // slice, len(list)
	if len(list) == 0 {
		return nil
	}
	switch list[0].type_ {
	case BSTR:
		for i, o := range list {
			val, err := o.Str()
			if err != nil {
				return err
			}
			v.Index(i).SetString(val)
		}
	case BINT:
		for i, o := range list {
			val, err := o.Int()
			if err != nil {
				return err
			}
			v.Index(i).SetInt(int64(val))
		}
	case BLIST:
		for i, o := range list {
			val, err := o.List()
			if err != nil {
				return err
			}
			// v is underneath p, and supposed to be a slice 
			if v.Type().Elem().Kind() != reflect.Slice {
				return ErrTyp
			}
			lp := reflect.New(v.Type().Elem()) // pointer: lp -> slice
			ls := reflect.MakeSlice(v.Type().Elem(), len(val), len(val)) // new slice with length and capacity equal to val(list)
			lp.Elem().Set(ls) // lp -> slice(ls)
			err = unmarshalList(lp, val) // recursivly call: fill the slice lp with elements from val
			if err != nil {
				return err
			}
			v.Index(i).Set(lp.Elem()) // v[slice(ls)]
			// why not set(lp)? v is a slice of sllice
		}
	case BDICT:
		for i, o := range list {
			val, err := o.Dict()
			if err != nil {
				return err
			}
			if v.Type().Elem().Kind() != reflect.Struct {
				return ErrTyp
			}
			dp := reflect.New(v.Type().Elem())
			// struct can set fields directly without a new empty struct like slice
			err = unmarshalDict(dp, val)
			if err != nil {
				return err
			}
			v.Index(i).Set(dp.Elem())
		}
	}
	return nil
}

func unmarshalDict(p reflect.Value, dict map[string]*BObject) error {
	if p.Kind() != reflect.Ptr || p.Elem().Type().Kind() != reflect.Struct {
		return errors.New("dest must be pointer")
	}
	v := p.Elem()
	for  i, n := 0, v.NumField(); i < n; i++ {
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
	} 
}

func Unmarshal(r io.Reader, s interface{}) error {
	o, err := Parse(r)
	if err != nil {
		return err
	}
	p := reflect.ValueOf(s)
	if p.Kind() != reflect.Ptr {
		return errors.New("dest must be a pointer")
	}
	switch o.type_ {
	case BLIST:
		list, _ := o.List() // -> [] *BObject
		// initialize a new empty slice, which has the same length as the BObject  
		l := reflect.MakeSlice(p.Elem().Type(), len(list), len(list))
		p.Elem().Set(l) // set each new slice to the container
		// why not append? there could be empty slices(p.elem())
		err = unmarshalList(p, list)
		if err != nil {
			return err
		}
	case BDICT:
		dict, _ := o.Dict()
		err = unmarshalDict(p, dict)
		
	}
}