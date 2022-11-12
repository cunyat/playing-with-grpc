package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func main() {
	flag.Parse()
	if flag.NArg() != 1 {
		log.Fatal("Expected exactly one argument with proto file")
	}

	file := flag.Arg(0)

	fd, err := loadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	srv := grpc.NewServer()
	streamer := &handler{}

	for i := 0; i < fd.Services().Len(); i++ {
		service := fd.Services().Get(i)

		desc := &grpc.ServiceDesc{
			ServiceName: string(service.FullName()),
			HandlerType: (*interface{})(nil),
		}

		for j := 0; j < service.Methods().Len(); j++ {
			method := service.Methods().Get(j)

			desc.Methods = append(desc.Methods, grpc.MethodDesc{
				MethodName: string(method.Name()),
				Handler:    methodHandler(method),
			})
		}

		fmt.Println(desc)
		srv.RegisterService(desc, streamer)
	}

	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}

	log.Fatal(srv.Serve(lis))
}

type handler struct{}

func (s *handler) handler(desc protoreflect.MethodDescriptor) grpc.StreamHandler {
	return func(srv interface{}, stream grpc.ServerStream) error {
		msg := dynamicpb.NewMessage(desc.Input())
		err := stream.RecvMsg(msg)
		if err != nil {
			return status.Error(codes.Internal, "could not receive msg: "+err.Error())
		}

		fmt.Printf("%+v\n", msg)
		return nil
	}
}

func methodHandler(desc protoreflect.MethodDescriptor) func(interface{}, context.Context, func(interface{}) error, grpc.UnaryServerInterceptor) (interface{}, error) {
	return func(srv interface{}, ctx context.Context, dec func(interface{}) error, _ grpc.UnaryServerInterceptor) (interface{}, error) {
		in := dynamicpb.NewMessage(desc.Input())
		if err := dec(in); err != nil {
			return nil, err
		}

		val := in.Get(desc.Input().Fields().ByName("name"))
		message := fmt.Sprintf("Hello %s", val.String())

		out := dynamicpb.NewMessage(desc.Output())
		out.Set(desc.Output().Fields().ByName("message"), protoreflect.ValueOf(message))

		return out, nil
	}
}

func loadFile(filename string) (protoreflect.FileDescriptor, error) {
	protoFile, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	pbSet := new(descriptorpb.FileDescriptorSet)
	if err := proto.Unmarshal(protoFile, pbSet); err != nil {
		return nil, err
	}

	// We know protoc was invoked with a single .proto file
	pb := pbSet.GetFile()[0]

	// Initialize the File descriptor object
	fd, err := protodesc.NewFile(pb, protoregistry.GlobalFiles)
	if err != nil {
		return nil, err
	}

	err = protoregistry.GlobalFiles.RegisterFile(fd)
	if err != nil {
		return nil, err
	}

	// and finally register it.
	return fd, nil
}
