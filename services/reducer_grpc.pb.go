// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.6.1
// source: services/reducer.proto

package services

import (
	context "context"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ReducerServiceClient is the client API for ReducerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ReducerServiceClient interface {
	SendIntermediateData(ctx context.Context, in *IntermediateData, opts ...grpc.CallOption) (*empty.Empty, error)
}

type reducerServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewReducerServiceClient(cc grpc.ClientConnInterface) ReducerServiceClient {
	return &reducerServiceClient{cc}
}

func (c *reducerServiceClient) SendIntermediateData(ctx context.Context, in *IntermediateData, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/services.ReducerService/SendIntermediateData", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ReducerServiceServer is the server API for ReducerService service.
// All implementations must embed UnimplementedReducerServiceServer
// for forward compatibility
type ReducerServiceServer interface {
	SendIntermediateData(context.Context, *IntermediateData) (*empty.Empty, error)
	mustEmbedUnimplementedReducerServiceServer()
}

// UnimplementedReducerServiceServer must be embedded to have forward compatible implementations.
type UnimplementedReducerServiceServer struct {
}

func (UnimplementedReducerServiceServer) SendIntermediateData(context.Context, *IntermediateData) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendIntermediateData not implemented")
}
func (UnimplementedReducerServiceServer) mustEmbedUnimplementedReducerServiceServer() {}

// UnsafeReducerServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ReducerServiceServer will
// result in compilation errors.
type UnsafeReducerServiceServer interface {
	mustEmbedUnimplementedReducerServiceServer()
}

func RegisterReducerServiceServer(s grpc.ServiceRegistrar, srv ReducerServiceServer) {
	s.RegisterService(&ReducerService_ServiceDesc, srv)
}

func _ReducerService_SendIntermediateData_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(IntermediateData)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ReducerServiceServer).SendIntermediateData(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/services.ReducerService/SendIntermediateData",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ReducerServiceServer).SendIntermediateData(ctx, req.(*IntermediateData))
	}
	return interceptor(ctx, in, info, handler)
}

// ReducerService_ServiceDesc is the grpc.ServiceDesc for ReducerService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ReducerService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "services.ReducerService",
	HandlerType: (*ReducerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SendIntermediateData",
			Handler:    _ReducerService_SendIntermediateData_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "services/reducer.proto",
}
