package services

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ReducerServer struct {
	UnimplementedReducerServiceServer
}

var reducerRootPath string

func (s *ReducerServer) SendIntermediateData(ctx context.Context, input *IntermediateData) (*emptypb.Empty, error) {
	log.Printf("Recieved intermediate data\n")
	filePath := reducerRootPath + "/" + input.FileName
	data, err := proto.Marshal(input.Data)
	if err != nil {
		log.Printf("Error serializing intermediate data: %v\n", err)
		return &emptypb.Empty{}, err
	}
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		log.Printf("Error writing serialized data: %v\n", err)
		return &emptypb.Empty{}, err
	}

	return &emptypb.Empty{}, nil
}

func InitReducerLogs() error {
	logFilePath := reducerRootPath + "/logs.txt"
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	log.Printf("-------------------------------------------------------------------------\n")
	// initialize logging
	log.SetOutput(logFile)
	return nil
}

func InitReducerFileSystem(port string) (error) {
	reducerRootPath = fmt.Sprintf("./reducers/r%s", port)
	if _, err := os.Stat(reducerRootPath); os.IsNotExist(err) {
		err := os.MkdirAll(reducerRootPath, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}

// create a connection pool
// connPool := []*grpc.ClientConn{}
// for i := 0; i < MasterConfig.Client.NMappers; i++ {
// 	mapperPort := MasterConfig.Mappers.Ports[i]
// 	var err error
// 	conn, err := grpc.Dial(fmt.Sprintf("localhost:%s", mapperPort), unsecureOpt, blockingOpt)
// 	connPool = append(connPool, conn)
// 	if err != nil {
// 		log.Printf("Error connection to the mapper on port: %s\n", mapperPort)
// 		return stream.SendAndClose(&emptypb.Empty{})
// 	}
// 	defer connPool[i].Close()
// }