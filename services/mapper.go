package services

import (
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// KvSorter sorter sorts key value pairs based on the key
type KvSorter []*KeyValue

func (k KvSorter) Len() int { return len(k) }
func (k KvSorter) Swap(i, j int) { k[i], k[j] = k[j], k[i] }
func (k KvSorter) Less(i, j int) bool { return k[i].Key < k[j].Key }

type MapperServer struct {
	UnimplementedMapperServiceServer
}

var mapperRootPath string

func (ms *MapperServer) RunMap(ctx context.Context, input *RunMapInput) (*emptypb.Empty, error) {
	log.Printf("Starting map function on the file: %s\n", input.FileName)
	var kvPairs *KvPairs
	// runs map function based on input
	log.Printf("Function: %s\n", input.Fn)
	if input.Fn == "wc" {
		kvPairs = wcMap(input.FileName, string(input.FileData))
	} else {
		kvPairs = invIndexMap(input.FileName, string(input.FileData))
	}

	log.Printf("Sorting intermediate key value pairs!\n")
	// sort kvPairs
	sort.Sort(KvSorter(kvPairs.Data))

	log.Printf("Map operation done!\n")

	nReducers := int(input.NReducers)
	// hash the pairs according to the reducer
	// hashing the word gives the bucket
	var reducerBuckets = map[int]*KvPairs{}
	for i := 0; i < nReducers; i++ {
		reducerBuckets[i] = &KvPairs{}
	}

	// bucket each pair
	log.Printf("Hashing keys into different buckets for reduce task\n")
	for _, pair := range kvPairs.Data {
		hsh := hashWordToBucket(pair.Key)
		bucket := hsh % nReducers
		reducerBuckets[bucket].Data = append(reducerBuckets[bucket].Data, pair)
	}

	// write buckets to intermediate files
	log.Printf("Writing intermediate files\n")
	for bucket := range reducerBuckets {
		data, err := proto.Marshal(reducerBuckets[bucket])
		if err != nil {
			log.Printf("Error seriazlizing data: %v\n", err)
			return &emptypb.Empty{}, err
		}
		bucketName := fmt.Sprintf("%s_task_%d_bucket_%d.bin", input.Fn, input.TaskId, bucket)
		bucketPath := fmt.Sprintf("%s/%s", mapperRootPath, bucketName)
		err = os.WriteFile(bucketPath, data, 0644)
		if err != nil {
			log.Printf("Error writing serialized data: %v\n", err)
			return &emptypb.Empty{}, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (ms *MapperServer) InitReduce(ctx context.Context, input *InitReduceInput) (*emptypb.Empty, error) {
	// read all the files from the mapperRootPath
	log.Printf("Starting init reduce\n")
	files, _ := os.ReadDir(mapperRootPath)
	bufferFiles := []string{}

	for _, file := range files {
		fileName := file.Name()
		if strings.Contains(fileName, "bucket") {
			log.Printf("Loading file %s\n", fileName)
			bufferFiles = append(bufferFiles, fileName)
		}
	}

	unsecureOpt := grpc.WithTransportCredentials(insecure.NewCredentials())
	blockingOpt := grpc.WithBlock()

	// TODO: add connection pooling
	// connPool := []*grpc.ClientConn{}
	// for _, port := range input.Ports {
	// 	log.Printf("Dialing reducer on port: %s\n", port)
	// 	conn, err := grpc.Dial(fmt.Sprintf("localhost:%s", port), unsecureOpt, blockingOpt)
	// 	connPool = append(connPool, conn)
	// 	if err != nil {
	// 		log.Printf("Error connection to reducer on port: %s\n", port)
	// 	}
	// 	// defer connPool[i].Close()
	// }

	log.Printf("Buffer Files: %d\n", len(bufferFiles))
	for _, fileName := range bufferFiles {
		bucket, _ := strconv.Atoi(string([]rune(fileName)[len(fileName) - 5])) // *_bucket.bin | *_0.bin
		log.Printf("Reading bucket %d files: %s\n", bucket, fileName)
		conn, err := grpc.Dial(fmt.Sprintf("localhost:%s", input.Ports[bucket]), unsecureOpt, blockingOpt)
		if err != nil {
			log.Printf("Error connection to reducer on port: %s\n", input.Ports[bucket])
			log.Printf("Error: %v\n", err)
			return &emptypb.Empty{}, err
		}
		defer conn.Close()
		rc := NewReducerServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
		defer cancel()

		data, err := os.ReadFile(mapperRootPath + "/" + fileName)
		if err != nil {
			log.Printf("Error reading intermediate file: %s\n", fileName)
			break
		}
		payload := &KvPairs{}
		err = proto.Unmarshal(data, payload)
		if err != nil {
			log.Printf("Error deserializing proto data of file: %s\n", fileName)
			break
		}

		log.Printf("Sending intermediate data to reducer at port: %s\n", input.Ports[bucket])
		_, err = rc.SendIntermediateData(ctx, &IntermediateData{FileName: fileName, Data: payload})
		if err != nil {
			log.Printf("Error sending intermediate data to the reducer at %s\n", input.Ports[bucket])
		}
		log.Printf("Sent intermediate data to reducer at port: %s\n", input.Ports[bucket])
	}

	return &emptypb.Empty{}, nil
}

func InitMapperFileSystem(port string) (error) {
	mapperRootPath = fmt.Sprintf("./mappers/m%s", port)
	// os.RemoveAll(mapperRootPath)
	err := os.MkdirAll(mapperRootPath, 0755)
	if err != nil {
		return err
	}
	return nil
}

func InitMapperLogs() error {
	logFilePath := fmt.Sprintf("%s/logs.txt", mapperRootPath)
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

// we do not use key in this function
func wcMap(_key, value string) *KvPairs {
	// spliting into words
	words := strings.FieldsFunc(value, func(r rune) bool { return !unicode.IsLetter(r) })
	// emit intermediate key value pairs
	kvPairs := &KvPairs{}
	for _, word := range words {
		kvPairs.Data = append(kvPairs.Data, &KeyValue{Key: word, Value: "1"})
	}

	return kvPairs
}

func invIndexMap(key, value string) *KvPairs {
	// spliting into words
	// key is the input file name
	words := strings.FieldsFunc(value, func(r rune) bool { return !unicode.IsLetter(r) })
	// emit intermediate key value pairs
	kvPairs := &KvPairs{}
	// generates word: fileName
	for _, word := range words {
		kvPairs.Data = append(kvPairs.Data, &KeyValue{Key: word, Value: key})
	}

	return kvPairs
}

func hashWordToBucket(word string) int {
	hFn := fnv.New32a()
	hFn.Write([]byte(word))
	return int(hFn.Sum32() & 0x7fffffff)
}