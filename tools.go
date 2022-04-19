//go:build tools
// +build tools

package main

import  (
		pb "github.com/golang/protobuf/protoc-gen-go"
		protoreflect "google.golang.org/protobuf/reflect/protoreflect"
		protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

func main() {
	_ := pb.error
	_ := protoreflect.const
	_ := protoimp.const
}