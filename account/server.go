// talks to both the client and the service
// calls functions from service and returns data to the client in gRPC format (gRPC protobuf data packets)
// PostAccount, GetAccount and GetAccounts take protobuf request and return protobuf response

package account

import (
	"context"
	"fmt"
	"net"

	"Microservices-based-E-commerce-System/account/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// struct that implements the methods from .proto service (PostAccount, GetAccount and GetAccounts)
type grpcServer struct {
	pb.UnimplementedAccountServiceServer // safety default implementation provided by gRPC. If one forgets to implement a method, the gRPC server will panic with a clear error instead of silently failing.
	service Service // talks to the repository
}

// starts gRPC server on a given port. Service is passed from main.go
func ListenGRPC(s Service, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port)) // Starts a TCP listener on the given port
	if err != nil {
		return err
	}
	serv := grpc.NewServer() // creates a new instance of a gRPC server which will handle incoming RPC calls
	pb.RegisterAccountServiceServer(serv, &grpcServer{
		UnimplementedAccountServiceServer: pb.UnimplementedAccountServiceServer{},
		service:                           s,
	}) // Registers AccountService with the gRPC server.
	reflection.Register(serv) // for debugging; lets clients discover services and methods dynamically at runtime.
	return serv.Serve(lis)    // starts the gRPC server and begins listening for requests on the TCP listener (lis).
}

// Receives PostAccountRequest, calls service.PostAccount, returns PostAccountResponse.
func (s *grpcServer) PostAccount(ctx context.Context, r *pb.PostAccountRequest) (*pb.PostAccountResponse, error) {
	a, err := s.service.PostAccount(ctx, r.Name)
	if err != nil {
		return nil, err
	}

	return &pb.PostAccountResponse{Account: &pb.Account{
		Id:   a.ID,
		Name: a.Name,
	}}, nil
}

func (s *grpcServer) GetAccount(ctx context.Context, r *pb.GetAccountRequest) (*pb.GetAccountResponse, error) {
	a, err := s.service.GetAccount(ctx, r.Id)
	if err != nil {
		return nil, err
	}
	return &pb.GetAccountResponse{Account: &pb.Account{
		Id:   a.ID,
		Name: a.Name,
	}}, nil
}

func (s *grpcServer) GetAccounts(ctx context.Context, r *pb.GetAccountsRequest) (*pb.GetAccountsResponse, error) {
	res, err := s.service.GetAccounts(ctx, r.Skip, r.Take)
	if err != nil {
		return nil, err
	}
	accounts := []*pb.Account{}
	for _, a := range res {
		accounts = append(accounts, &pb.Account{
			Id:   a.ID,
			Name: a.Name,
		})
	}

	return &pb.GetAccountsResponse{Accounts: accounts}, nil
}
