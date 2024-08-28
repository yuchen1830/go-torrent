package bencode

import (
	"errors"
	"io"
	"reflect"
	"strings"
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
	// struct: loop all fields
	for  i, n := 0, v.NumField(); i < n; i++ {
		fv := v.Field(i)
		// accesses to the value of the i-th field, represents the value
		if !fv.CanSet() {
			continue
		}
		ft := v.Type().Field(i)
		// accesses to the metadata of the i-th field, which contains multiple attributes
		key := ft.Tag.Get("bencode") // check the Tag first, otherwise make sure the key in struct is public
		if key == "" {
			key = strings.ToLower(ft.Name)
		}
		fo := dict[key] // *BObject
		if fo == nil {
			continue
		}
		//  to provide the data that will be assigned to the struct's field
		switch fo.type_ {
		case BSTR:
			if ft.Type.Kind() != reflect.String {
				break
			}
			val, _ := fo.Str()
			fv.SetString(val)
		case BINT:
			if ft.Type.Kind() != reflect.Int{
				break
			}
			val, _ := fo.Int()
			fv.SetInt(int64(val))
		case BLIST:
			if ft.Type.Kind() != reflect.Slice {
				break
			}
			// to ensure that the value being set matches the type of the field
			list, _ := fo.List()
			lp := reflect.New(ft.Type)
			ls := reflect.MakeSlice(ft.Type, len(list), len(list))
			lp.Elem().Set(ls)
			err := unmarshalList(lp, list)
			if err != nil {
				break
			}
			fv.Set(lp.Elem())
		case BDICT:
			if ft.Type.Kind() != reflect.Struct {
				break
			}
			dp := reflect.New(ft.Type)
			dict, _ := fo.Dict()
			err := unmarshalDict(dp, dict)
			if err != nil {
				break
			}
			fv.Set(dp.Elem())
		}
	} 
	return nil
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
		if err != nil {
			return err
		}
	default:
		return errors.New("src code must be struct or slice")
	}
	return nil
}

// basic: encode
func marshalValue(w io.Writer, v reflect.Value) int {
	len := 0
	switch v.Kind() {
	case reflect.String:
		len += EncodeString(w, v.String())
	case reflect.Int:
		len += EncodeInt(w, int(v.Int()))
	case reflect.Slice:
		len += marshalList(w, v)
	case reflect.Struct:
		len += marshalDict(w, v)
	}
	return len
}

// l -- e
func marshalList(w io.Writer, v reflect.Value) int {
	len := 2
	w.Write([]byte{'l'})
	for i := 0; i < v.Len(); i++ {
		ev := v.Index(i)
		// marshal the nested elements
		len += marshalValue(w, ev)
	}
	w.Write([]byte{'e'})
	return len
}

// d -- e
func marshalDict(w io.Writer, v reflect.Value) int {
	len := 2
	w.Write([]byte{'l'})
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		ft := v.Type().Field(i)
		key := ft.Tag.Get("bencode")
		if key == "" {
			key = strings.ToLower(ft.Name)
		}
		// marshal the nested elements
		len += EncodeString(w, key)
		len += marshalValue(w, fv)
	}
	w.Write([]byte{'e'})
	return len
}

// struct / slice -> bencode
func Marshal(w io.Writer, s interface{}) int {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return marshalValue(w, v)
}