// Code generated by protoc-gen-go-json. DO NOT EDIT.
// source: hapi/release/info.proto

package release

import (
	"bytes"

	"github.com/golang/protobuf/jsonpb"
)

// MarshalJSON implements json.Marshaler
func (msg *Info) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	err := (&jsonpb.Marshaler{
		EnumsAsInts:  false,
		EmitDefaults: false,
		OrigName:     false,
	}).Marshal(&buf, msg)
	return buf.Bytes(), err
}

// UnmarshalJSON implements json.Unmarshaler
func (msg *Info) UnmarshalJSON(b []byte) error {
	return (&jsonpb.Unmarshaler{
		AllowUnknownFields: false,
	}).Unmarshal(bytes.NewReader(b), msg)
}
