package services

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

var masterRootPath string

// cluster config from config file
type Config struct {
	Client struct {
		NMappers int `json:"nMappers"`
		NReducers int `json:"nReducers"`
	}
	Master struct {
		Port string `json:"port"`
	} `json:"master"`
	Mappers struct {
		MaxAllowed int `json:"maxAllowed"`
		Ports []string `json:"ports"`
	} `json:"mappers"`
	Reducers struct {
		MaxAllowed int `json:"maxAllowed"`
		Ports []string `json:"ports"`
	} `json:"reducers"`
}

var MasterConfig Config

type MasterServer struct {
	UnimplementedMasterServiceServer
}

func (s *MasterServer) InitCluster(ctx context.Context, input *IcInput) (*emptypb.Empty, error) {
	// init mappers and reducers
	for i := 0; i < int(input.NMappers); i++ {
		cmd := exec.Command("go", "run", "main.go", "mapper", MasterConfig.Mappers.Ports[i])
		err := cmd.Start()
		if err != nil {
			return &emptypb.Empty{}, err
		}
	}

	for i := 0; i < int(input.NReducers); i++ {
		cmd := exec.Command("go", "run", "main.go", "reducer", MasterConfig.Reducers.Ports[i])
		err := cmd.Start()
		if err != nil {
			return &emptypb.Empty{}, err
		}
	}
	
	return &emptypb.Empty{}, nil
}

func (s *MasterServer) RunMapRd(stream MasterService_RunMapRdServer) (error) {
	// listen to the stream
	fn := ""
	for {
		input, err := stream.Recv()
		// read function type
		if len(fn) == 0 {
			fn = input.Fn
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		err = os.WriteFile(masterRootPath + "/input_" + input.File.Name, input.File.Data, 0655)
		if err != nil {
			log.Printf("Error writing input files: %v\n", err)
			return err
		}
	}

	// send the files to each map job
	// for each file
	files, err := os.ReadDir(masterRootPath)
	if err != nil {
		log.Printf("Error reading root: %v\n", err)
		return stream.SendAndClose(&emptypb.Empty{})
	}

	var wg sync.WaitGroup
	unsecureOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	blockingOpt := grpc.WithBlock()

	inputFiles := []fs.DirEntry{}
	for _, file := range files {
		fileName := file.Name()
		if strings.Contains(fileName, "input") {
			inputFiles = append(inputFiles, file)
		}
	}
	
	for i, file := range inputFiles {
		// at max we can send files to 1 mapper at a time
		mapperIndex := i % MasterConfig.Client.NMappers
		log.Printf("Sending task %d to mapper", i)
		wg.Add(1)
		go func(i int) {
			// create a connection
			mapperPort := MasterConfig.Mappers.Ports[mapperIndex]
			conn, err := grpc.Dial(fmt.Sprintf("localhost:%s", mapperPort), unsecureOpt, blockingOpt)
			if err != nil {
				// mapper connection failed
				// maybe its down?
				// handle faults here
				log.Print("Error: ", err)
				return
			}
			
			defer conn.Close()
			defer wg.Done()
			
			mc := NewMapperServiceClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
			defer cancel()

			fileData, err := os.ReadFile(masterRootPath + "/" + file.Name())
			if err != nil {
				log.Printf("Error reading filedata: %s\n", file.Name())
				return
			}
			runMapInput := &RunMapInput{
				TaskId: int32(i),
				NReducers: int32(MasterConfig.Client.NReducers),
				Fn: fn,
				FileName: file.Name(),
				FileData: fileData,
			}
			_, err = mc.RunMap(ctx, runMapInput)
			if err != nil {
				// map job failed, handle fault
				log.Print("Error: ", err)
			}
		}(i)

		if mapperIndex + 1 == MasterConfig.Client.NMappers {
			wg.Wait()
		}
	}

	wg.Wait()

	// all map tasks are done
	log.Printf("All map tasks are done!\n")
	
	// start init reduce tasks
	// send the intermediate data to reducers
	log.Printf("Starting reduce tasks\n")
	for i := 0; i < MasterConfig.Client.NMappers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			mapperPort := MasterConfig.Mappers.Ports[i]
			conn, err := grpc.Dial(fmt.Sprintf("localhost:%s", mapperPort), unsecureOpt, blockingOpt)
			if err != nil {
				log.Printf("Error: %v\n", err)
				return
			}
			defer conn.Close()

			mc := NewMapperServiceClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
			defer cancel()

			_, err = mc.InitReduce(ctx, &InitReduceInput{Ports: MasterConfig.Reducers.Ports})
			if err != nil {
				log.Printf("Error starting InitReduce on mapper port: %s\n", MasterConfig.Mappers.Ports[i])
				log.Printf("Error: %v\n", err)
			}
		}(i)
	}

	wg.Wait()

	return stream.SendAndClose(&emptypb.Empty{})
}

func InitMasterLogs() error {
	logFilePath := masterRootPath + "/logs.txt"
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	log.Printf("-------------------------------------------------------------------------\n")
	// initialize logging
	log.SetOutput(logFile)
	return nil
}

func InitMasterFileSystem() (error) {
	masterRootPath = "./master"
	if _, err := os.Stat(masterRootPath); os.IsNotExist(err) {
		err := os.MkdirAll(masterRootPath, 0755)
		if err != nil {
			return err
		}
	}
	return nil
}