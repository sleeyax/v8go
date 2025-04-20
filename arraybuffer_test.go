package v8go

import (
	"errors"
	"fmt"
	"testing"
)

func reverseArrayBufferFunctionCallback(info *FunctionCallbackInfo) *Value {
	iso := info.Context().Isolate()
	args := info.Args()
	if len(args) != 1 {
		return iso.ThrowExceptionStringUnstable("Function ReverseArrayBuffer expects 1 parameter")
	}
	if !args[0].IsArrayBuffer() {
		return iso.ThrowExceptionStringUnstable("Function ReverseArrayBuffer expects ArrayBuffer parameter")
	}
	ab := args[0].ArrayBuffer() // "cast" to ArrayBuffer
	length := int(ab.ByteLength())
	bytes := ab.GetBytes() // get a copy of the bytes from the ArrayBuffer
	reversed := make([]uint8, length)
	for i := 0; i < length; i++ {
		reversed[i] = bytes[length-i-1]
	}
	ab.PutBytes(reversed) // update the bytes in the ArrayBuffer (length must match!)
	return nil
}

func createArrayBufferFunctionCallback(info *FunctionCallbackInfo) *Value {
	iso := info.Context().Isolate()
	args := info.Args()
	if len(args) != 1 {
		return iso.ThrowExceptionStringUnstable("Function CreateArrayBuffer expects 1 parameter")
	}
	if !args[0].IsInt32() {
		return iso.ThrowExceptionStringUnstable("Function CreateArrayBuffer expects Int32 parameter")
	}
	length := args[0].Int32()
	ab := NewArrayBuffer(info.Context(), int64(length)) // create ArrayBuffer object of given length
	bytes := make([]uint8, length)
	for i := uint8(0); i < uint8(length); i++ {
		bytes[i] = i
	}
	ab.PutBytes(bytes) // copy these bytes into it. Caller is responsible for avoiding overruns!
	return ab.Value    // return the ArrayBuffer to javascript
}

func injectArrayBufferTester(ctx *Context, funcName string, funcCb FunctionCallback) error {
	if ctx == nil {
		return errors.New("injectArrayBufferTester: ctx is required")
	}

	iso := ctx.Isolate()

	con := NewObjectTemplate(iso)

	funcTempl := NewFunctionTemplate(iso, funcCb)

	if err := con.Set(funcName, funcTempl, ReadOnly); err != nil {
		return fmt.Errorf("injectArrayBufferTester: %v", err)
	}

	nativeObj, err := con.NewInstance(ctx)
	if err != nil {
		return fmt.Errorf("injectArrayBufferTester: %v", err)
	}

	global := ctx.Global()

	if err := global.Set("native", nativeObj); err != nil {
		return fmt.Errorf("injectArrayBufferTester: %v", err)
	}

	return nil
}

// Test that a script can call a go function to reverse an ArrayBuffer.
// The function reverses the ArrayBuffer in-place, i.e. this is a call-by-reference.
func TestModifyArrayBuffer(t *testing.T) {
	t.Parallel()

	iso := NewIsolate()
	ctx := NewContext(iso)
	if err := injectArrayBufferTester(ctx, "reverseArrayBuffer", reverseArrayBufferFunctionCallback); err != nil {
		t.Error(err)
	}

	js := `
		let ab = new ArrayBuffer(10);
		let view = new Uint8Array(ab);
		for (let i = 0; i < 10; i++) view[i] = i;
		native.reverseArrayBuffer(ab);
		ab;
	`

	if val, err := ctx.RunScript(js, ""); err != nil {
		t.Error(err)
	} else {
		if !val.IsArrayBuffer() {
			t.Errorf("Expected ArrayBuffer return value")
		}
		ab := val.ArrayBuffer()
		if ab.ByteLength() != 10 {
			t.Errorf("Got wrong ArrayBuffer length %d, expected 10", ab.ByteLength())
		}
		bytes := ab.GetBytes()
		fmt.Printf("Got reversed ArrayBuffer from script: %v\n", bytes)
		for i := int64(0); i < ab.ByteLength(); i++ {
			if bytes[i] != uint8(10-i-1) {
				t.Errorf("Incorrect byte at index %d (whole array: %v)", i, ab)
			}
		}
	}
}

func TestCreateArrayBuffer(t *testing.T) {
	t.Parallel()

	iso := NewIsolate()
	ctx := NewContext(iso)
	if err := injectArrayBufferTester(ctx, "createArrayBuffer", createArrayBufferFunctionCallback); err != nil {
		t.Error(err)
	}

	js := `
		native.createArrayBuffer(16);
	`

	if val, err := ctx.RunScript(js, ""); err != nil {
		t.Error(err)
	} else {
		if !val.IsArrayBuffer() {
			t.Errorf("Expected ArrayBuffer return value")
		}
		ab := val.ArrayBuffer()
		if ab.ByteLength() != 16 {
			t.Errorf("Got wrong ArrayBuffer length %d, expected 16", ab.ByteLength())
		}
		bytes := ab.GetBytes()
		fmt.Printf("Got ArrayBuffer from script: %v\n", bytes)
		for i := int64(0); i < ab.ByteLength(); i++ {
			if bytes[i] != uint8(i) {
				t.Errorf("Incorrect byte at index %d (whole array: %v)", i, bytes)
			}
		}
	}
}
