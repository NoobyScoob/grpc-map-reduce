package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"time"

	"github.com/noobyscoob/grpc-map-reduce/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// global config variable
var config = services.Config{}

func main() {
	loadDefaultConfig()
	// function name
	if os.Args[1] == "client" {
		startRpcClient()
	} else if os.Args[1] == "master" {
		startRpcServer(config.Master.Port)
	} else {
		// os.Args[2] contains worker type
		startRpcServer(os.Args[2])
	}
}

// ./main client inputFilesPath operation
// ./main master 
// starts the master and run initCluster?
// how do you know when the master is up?

func startRpcClient() {
	cmd := exec.Command("go", "run", "main.go", "master")
	// just for this local map reduce
	out, _ := cmd.StdoutPipe()
	defer out.Close()

	// clean file system
	os.RemoveAll("./master")
	os.RemoveAll("./mappers")
	os.RemoveAll("./reducers")
	os.RemoveAll("./output")
	
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	log.Print(cmd.Process.Pid)
	log.Print(config.Master.Port)

	// log.Printf("Starting a master process on port: %s!\nProcess Id: %d", config.Master.Port, proc.Pid)

	unsecureOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	blockingOpt := grpc.WithBlock()

	var conn *grpc.ClientConn
	conn, err = grpc.Dial(fmt.Sprintf("localhost:%s", config.Master.Port), unsecureOpt, blockingOpt)
	if err != nil {
		log.Print("Error: ", err)
	}

	defer conn.Close()

	mc := services.NewMasterServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 60 * time.Second)
	defer cancel()

	inputFilesPath := os.Args[2]
	fn := os.Args[3]

	log.Printf("Number of mappers (can be updated in config.json): %d\n", config.Client.NMappers)
	log.Printf("Number of reducers (can be updated in config.json): %d\n", config.Client.NReducers)
	log.Printf("Input files are located at: %s\n", inputFilesPath)
	log.Printf("Running function (wc/ii): %s\n", fn)

	log.Printf("Initializing cluster...\n")

	_, err = mc.InitCluster(ctx, &services.IcInput{
		NMappers: int32(config.Client.NMappers),
		NReducers: int32(config.Client.NReducers),
	})
	if err != nil {
		log.Fatal(err)
	}

	inputFiles, err := os.ReadDir(inputFilesPath)
	if err != nil {
		log.Fatal("Read Dir", err)
	}

	log.Printf("Check log files in ./master, ./mappers and ./reducers folders\n")
	log.Printf("Running map reduce...\n")

	stream, err := mc.RunMapRd(context.Background())
	if err != nil {
		log.Fatal("Stream creation error", err)
	}

	for _, file := range inputFiles {
		log.Printf(file.Name())
		bytes, err := os.ReadFile(inputFilesPath + file.Name())
		if err != nil {
			log.Fatal("Read file err ", err)
		}
		payload := &services.RunMapRdInput{Fn: fn, File: &services.FileInput{Name: file.Name(), Data: bytes}}
		err = stream.Send(payload)
		if err != nil {
			log.Fatal("Stream Send ", err)
		}
	}

	// when this is done all the map reduce jobs are done
	_, err = stream.CloseAndRecv()
	if err != nil {
		log.Fatal("Close and Recv ", err)
	}
}

func startRpcServer(port string) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	log.Printf("RPC server listening on port %s", port)

	if err != nil {
		log.Fatalf("Listenting on port %s failed: %v\n", port, err)
	}

	grpcServer := grpc.NewServer()

	switch workerType := os.Args[1]; workerType {
	case "master":
		services.MasterConfig = config
		services.InitMasterFileSystem()
		// logs are initalized after file system creation only
		services.InitMasterLogs()
		master := services.MasterServer{}
		services.RegisterMasterServiceServer(grpcServer, &master)
		log.Printf("Registered master service on port: %s\n", config.Master.Port)
	case "mapper":
		services.InitMapperFileSystem(os.Args[2])
		// logs are initalized after file system creation only
		services.InitMapperLogs()
		mapper := services.MapperServer{}
		services.RegisterMapperServiceServer(grpcServer, &mapper)
		log.Printf("Registered mapper service on port: %s\n", os.Args[2])
	case "reducer":
		services.InitReducerFileSystem(os.Args[2])
		// logs are initalized after file system creation only
		services.InitReducerLogs()
		reducer := services.ReducerServer{}
		services.RegisterReducerServiceServer(grpcServer, &reducer)
		log.Printf("Registered reducer service on port: %s\n", os.Args[2])
	}

	grpcServer.Serve(listener)
}

func loadDefaultConfig() {
	bytes, _ := os.ReadFile("./config.json")
	json.Unmarshal(bytes, &config)
}