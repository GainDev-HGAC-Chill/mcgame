// ABC parsing locates code by owning class and trait, never by global numeric replacement.
package main

import (
	"encoding/binary"
	"fmt"
)

type abcReader struct {
	data    []byte
	pos     int
	err     error
	strings []string
	names   []string
	owners  map[uint32]string
}

func (r *abcReader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if n < 0 || n > len(r.data)-r.pos {
		r.err = fmt.Errorf("truncated ABC at %d", r.pos)
		return nil
	}
	v := r.data[r.pos : r.pos+n]
	r.pos += n
	return v
}

func (r *abcReader) byte() byte {
	v := r.take(1)
	if len(v) == 0 {
		return 0
	}
	return v[0]
}

func (r *abcReader) u30() uint32 {
	var value uint32
	for i := 0; i < 5; i++ {
		b := r.byte()
		value |= uint32(b&127) << uint(7*i)
		if b&128 == 0 {
			return value
		}
	}
	r.err = fmt.Errorf("invalid ABC integer")
	return 0
}

func (r *abcReader) text(index uint32) string {
	if int(index) >= len(r.strings) {
		r.err = fmt.Errorf("invalid string index")
		return ""
	}
	return r.strings[index]
}

func (r *abcReader) name(index uint32) string {
	if int(index) >= len(r.names) {
		r.err = fmt.Errorf("invalid multiname index")
		return ""
	}
	return r.names[index]
}

func (r *abcReader) pool() {
	r.take(4)
	for j := 0; j < 2; j++ {
		n := r.u30()
		for i := uint32(1); i < n; i++ {
			r.u30()
		}
	}
	n := r.u30()
	if n > 0 {
		r.take(int(n-1) * 8)
	}
	n = r.u30()
	r.strings = []string{""}
	for i := uint32(1); i < n; i++ {
		size := r.u30()
		r.strings = append(r.strings, string(r.take(int(size))))
	}
	n = r.u30()
	namespaces := []string{""}
	for i := uint32(1); i < n; i++ {
		r.byte()
		namespaces = append(namespaces, r.text(r.u30()))
	}
	n = r.u30()
	for i := uint32(1); i < n; i++ {
		count := r.u30()
		for j := uint32(0); j < count; j++ {
			r.u30()
		}
	}
	n = r.u30()
	r.names = []string{""}
	for i := uint32(1); i < n; i++ {
		name := ""
		switch r.byte() {
		case 0x07, 0x0d:
			ns := r.u30()
			name = r.text(r.u30())
			if int(ns) >= len(namespaces) {
				r.err = fmt.Errorf("invalid namespace")
			} else if namespaces[ns] != "" {
				name = namespaces[ns] + "." + name
			}
		case 0x0f, 0x10:
			name = r.text(r.u30())
		case 0x11, 0x12:
		case 0x09, 0x0e:
			name = r.text(r.u30())
			r.u30()
		case 0x1b, 0x1c:
			r.u30()
		case 0x1d:
			r.u30()
			count := r.u30()
			for j := uint32(0); j < count; j++ {
				r.u30()
			}
		default:
			r.err = fmt.Errorf("unsupported multiname kind")
		}
		r.names = append(r.names, name)
		if r.err != nil {
			return
		}
	}
}

func (r *abcReader) traits(owner string) {
	n := r.u30()
	for i := uint32(0); i < n; i++ {
		name := r.name(r.u30())
		kind := r.byte()
		switch kind & 15 {
		case 0, 6:
			r.u30()
			r.u30()
			if r.u30() != 0 {
				r.byte()
			}
		case 1, 2, 3:
			r.u30()
			method := r.u30()
			if owner != "" {
				r.owners[method] = owner + ":" + name
			}
		case 4:
			r.u30()
			r.u30()
		case 5:
			r.u30()
			r.u30()
		default:
			r.err = fmt.Errorf("unsupported trait kind")
		}
		if kind&0x40 != 0 {
			count := r.u30()
			for j := uint32(0); j < count; j++ {
				r.u30()
			}
		}
		if r.err != nil {
			return
		}
	}
}

func (r *abcReader) definitions() {
	n := r.u30()
	for i := uint32(0); i < n; i++ {
		params := r.u30()
		r.u30()
		for j := uint32(0); j < params; j++ {
			r.u30()
		}
		r.u30()
		flags := r.byte()
		if flags&8 != 0 {
			count := r.u30()
			for j := uint32(0); j < count; j++ {
				r.u30()
				r.byte()
			}
		}
		if flags&128 != 0 {
			for j := uint32(0); j < params; j++ {
				r.u30()
			}
		}
	}
	n = r.u30()
	for i := uint32(0); i < n; i++ {
		r.u30()
		count := r.u30()
		for j := uint32(0); j < count*2; j++ {
			r.u30()
		}
	}
	n = r.u30()
	classes := make([]string, n)
	for i := uint32(0); i < n; i++ {
		classes[i] = r.name(r.u30())
		r.u30()
		flags := r.byte()
		if flags&8 != 0 {
			r.u30()
		}
		count := r.u30()
		for j := uint32(0); j < count; j++ {
			r.u30()
		}
		r.owners[r.u30()] = classes[i] + ":constructor"
		r.traits(classes[i])
	}
	for i := uint32(0); i < n; i++ {
		r.owners[r.u30()] = classes[i] + ":initializer"
		r.traits(classes[i])
	}
	n = r.u30()
	for i := uint32(0); i < n; i++ {
		r.u30()
		r.traits("")
	}
}

func patchABC(data []byte, counts map[string]int) error {
	if len(data) < 4 || binary.LittleEndian.Uint16(data[2:]) != 46 {
		return fmt.Errorf("unsupported ABC version")
	}
	r := &abcReader{data: data, owners: make(map[uint32]string)}
	r.pool()
	if r.err != nil {
		return r.err
	}
	r.definitions()
	if r.err != nil {
		return r.err
	}
	n := r.u30()
	for i := uint32(0); i < n; i++ {
		method := r.u30()
		for j := 0; j < 4; j++ {
			r.u30()
		}
		length := r.u30()
		code := r.take(int(length))
		if r.err != nil {
			return r.err
		}
		if err := patchCode(r.owners[method], code, r.names, r.strings, counts); err != nil {
			return err
		}
		exceptions := r.u30()
		for j := uint32(0); j < exceptions; j++ {
			for k := 0; k < 5; k++ {
				r.u30()
			}
		}
		r.traits("")
		if r.err != nil {
			return r.err
		}
	}
	if r.pos != len(data) {
		return fmt.Errorf("unparsed ABC bytes at %d/%d", r.pos, len(data))
	}
	return nil
}
