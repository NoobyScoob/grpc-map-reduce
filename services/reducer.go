package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ReducerServer struct {
	UnimplementedReducerServiceServer
}

var reducerRootPath string
var runningPort string

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

func (s *ReducerServer) RunReduce(ctx context.Context, input *RunReduceInput) (*FileOutput, error) {
	log.Printf("Starting redue task!\n")
	// read all intermediate files
	groupedData := make(map[string][]string)

	// input files, read how?
	// same as mapper
	files, _ := os.ReadDir(reducerRootPath)
	bufferFiles := []string{}

	// after every reduce task is complete
	// file system is cleaned except for logs
	for _, file := range files {
		fileName := file.Name()
		if strings.Contains(fileName, "bucket") {
			log.Printf("Loading file %s\n", fileName)
			bufferFiles = append(bufferFiles, fileName)
		}
	}

	var wg sync.WaitGroup
	kvChan := make(chan *KeyValue, 1000)

	for _, fileName := range bufferFiles {
		// multiple threads are spawned to
		// read files and sends data to the kvChannel
		wg.Add(1)
		go func(fileName string) {
			defer wg.Done()

			log.Printf("Reading buffer file: %s\n", fileName)
			fileData, err := os.ReadFile(reducerRootPath + "/" + fileName)
			if err != nil {
				log.Printf("Error reading intermediate file: %s\n", fileName)
				return
			}

			log.Printf("Deserializing buffer file: %s\n", fileName)
			data := &KvPairs{}
			err = proto.Unmarshal(fileData, data)
			if err != nil {
				log.Printf("Error deserializing the data: %v\n", err)
				return
			}
			log.Printf("Streaming kv pairs to grouping thread!\n")
			for _, kv := range data.Data {
				kvChan <- kv
			}
		}(fileName)
	}

	// a go routine that listens on the kvChan to group data
	log.Printf("Registering a thread to group data\n")
	go func() {
		for kv := range kvChan {
			_, ok := groupedData[kv.Key]
			if ok {
				// key present! append data
				groupedData[kv.Key] = append(groupedData[kv.Key], kv.Value)
			} else {
				groupedData[kv.Key] = []string{kv.Value}
			}
		}
	}()

	wg.Wait()
	close(kvChan)

	log.Printf("Groupby operation complete!\n")
	log.Printf("Writing result to out.txt file...on %s\n", runningPort)
	outFileName := fmt.Sprintf("out%s.txt", runningPort)
	outFilePath := fmt.Sprintf("%s/%s", reducerRootPath, outFileName)
	file, _ := os.OpenFile(outFilePath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	
	var out string
	for k, v := range groupedData {
		if input.Fn == "wc" {
			out = wcReduce(k, v)
		} else {
			out = invIndexReduce(k, v)
		}
		_, err := file.WriteString(fmt.Sprintf("%s: %s\n", k, out))
		if err != nil {
			log.Printf("Error writing output: %v\n", err)
		}
	}

	bytes, _ := os.ReadFile(outFilePath)
	return &FileOutput{Name: outFileName, Data: bytes}, nil
}

func InitReducerLogs() error {
	logFilePath := reducerRootPath + "/logs.txt"
	logFile, err := os.OpenFile(logFilePath, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	log.Printf("-------------------------------------------------------------------------\n")
	// initialize logging
	multi := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multi)
	return nil
}

func InitReducerFileSystem(port string) (error) {
	runningPort = port
	reducerRootPath = fmt.Sprintf("./reducers/r%s", port)
	// os.RemoveAll(reducerRootPath)
	err := os.MkdirAll(reducerRootPath, 0755)
	if err != nil {
		return err
	}
	return nil
}

func wcReduce(key string, values []string) string {
	sum := 0
	for i := 0; i < len(values); i++ {
		intVal, err := strconv.Atoi(values[i])
		if err != nil {
			log.Printf("Error converting value: %s\n", values[i])
		}
		sum += intVal
	}

	return strconv.Itoa(sum)
}

func invIndexReduce(key string, values []string) string {
	// sort the strings to make it easier to generate
	// unique file names
	sort.Strings(values)
	out := []string{}
	lastOut := ""
	for _, value := range values {
		if lastOut != value {
			// removing "input_" from file name
			out = append(out, value[6:])
			lastOut = value
		}
	}
	return fmt.Sprintf("%d %s", len(out), strings.Join(out, ","))
}