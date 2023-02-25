# MapReduce with gRpc!

This assignment is an implementation of MapReduce framework in Golang using gRpc (Remote procedure call framework by Google). It can compute operations on large dataset using distributed setting, while achieving parallelism and concurrency.

**Key Words and Phrases** : Golang, RPC, gRpc, Protocol Buffers.

## 1. Introduction

In this implementation of MapReduce in, master spawns the map reduce processes on one a single machine as OS processes. They communicate using the _gRpc_ implementation of remote procedure calls (which underlyingly use Http 2.0). Current design tests the implementation on two operations, _wordcount_ and _inverted index._ This also uses better serializing mechanism, with _protocol buffers_, achieves parallelism of connections and concurrency in processing of map and reduce tasks with goroutines (also called threads).

## 2. Design

## 3. Implementation

### 3.1 Client

**File** : main.go

Client is a user program that initiates the map reduce process, i.e., starts the master with user specific configurations. In this implementation immediately after starting the client process with the required input, it creates a separate master process. Client listens on the connection until the jobs are processed and gets notified about the output.

**Input:** input files, type of operation, ports configuration, number of mappers and reducers.

**Output:** output readable files

**Command** : $go run main.go client ./input/filepath/ wc

Arguments

- Program type (client)
- Input files path (must be given as format "./path/". Ends with /)
- Function type (wc/ii)

### 3.2 Master

**File** : master.go

**Protocol Defination** : master.proto

Master program is started by the client in a _separate OS process_ which listens for RPC connections. After client establishes the connection with the master it calls the initialize map reduce function to spin up mappers and reducers respectively.

- Master uses configuration file (config.json) to load number of mappers and reduces.
- Master and client maintains connection stream to notify the client.
- After successful initialization of mapper and reducer processes. Client starts the map reduce task by initiating RPC call to the master.

### 3.3 Mapper

**File** : mapper.go

**Protocol Defination** : mapper.proto

Each mapper process spawned by the master takes one input file from the master and generates intermediate files.

- Mapper calls the map function given by the user as input to the client program.
- Mapper hashes each word with a custom hash function (32-bit FNV-1a Hash).
- Mapper **sorts** the resultant key value pairs.
- Buckets of intermediate data according to the number of reducers are created.
  - (Hash output) % number of Reducers
- Intermediate files are stored as **protocol buffers**
  - compared to JSON or any other human readable formats is better because it is a serialized binary file.
- At the end all the mappers notify the master accordingly.
- After all mappers notify the master. Master initiates an RPC call to the master where each master sends the intermediate binary files to the reducer buckets with respect to the hash function.

### 3.4 Reducer

**File** : reducer.go

**Protocol Defination** : reducer.proto

All mappers and reducers are started at the same time. So, the reducers are waiting idle listening for intermediate files from the mappers.

- Each reducer receives intermediate _protocol buffers_ from all the mappers.
- Reducers first store the files received in their local storage.
- After all reducers receive the intermediate files from mappers. Each mapper notifies the master.
- Master initiates the run reduce call on each reducer.
- Each reducer groups the intermediate data from all files and runs the reduce function.
- Results of the reduce function are stored in output files and sent to the master.

### 3.5 Map & Reduce functions

Implementation of Word Count and Inverted Index are provided. 
Example:

Respective functions are implemented in mapper.go and reducer.go files.

### 3.6 Distributed Group by

Grouping implementation is split into two stages where:

- Mapper sorts all the keys to make it easier to run the group by task later.
- Each reducer groups the keys and runs the reduce function.

### 3.7 Parallelism and Concurrency

- All the tasks to mappers are sent by the master concurrently using threads.
- Each mapper runs their respective task and achieves parallelism.
- Grouping is done concurrently at the reducer using threads (goroutines), using the fact that we have multiple intermediate files from multiple masters.
- Mapper writes the intermediate binary files using threads.

### 3.8 Directory Structure

- Client
  - At root: _main.go_
  - Input files: _input/small and input/large_
- Mappers
  - Implementation: _services/mapper.go, servicers/mapper.proto_
  - Separate directory for each mapper
  - Ex: mappers/m9091 (number is port)
  - txt (for logs)
  - Contains intermediate serialized binary files
- Reducers
  - Implementation: _services/reducer.go, services/reducer.proto_
  - Separate directory for each reducer
  - Ex: reducers/r9091 (number is port)
  - txt (for logs)
  - Contains intermediate serialized binary files
- Master
  - Implementation: _services/master.go, services/mapper.proto_
  - Separate directory
  - Ex: master/
  - Contains text input files
- Config
  - json file (for mapper ports, reducer ports e.t.c.)
- Executables (after running make)
  - ./bin/main\_darwin for macs
  - ./bin/main\_linux for linux
- Output (in ./output)

# 4. Build, Run & Tests

### 4.1 Makefile

_$make_: Builds the binary files.
_$make test_: runs word count and inverted index tests
_$make run_: builds and runs the program

## 4.2 Environment and Execution

Go version 1.20 is used to develop.
Install the latest go version from here: https://go.dev/doc/install

**Manual Execution:**

After running each test case use below command to cleanup ports.
$killall main

Word Count:

Test1: $go run main.go client ./input/small/ wc
Test2: go run main.go client ./input/large/ wc

Inverted Index:

Test1: $go run main.go client ./input/small/ ii
Test2: go run main.go client ./input/large/ ii

Can use `$./bin/main_linux` instead of `$go run main.go`

**Known Edge Cases:** unsupported characters in the text file, large input files (\>5mb), not closed connections and files.

## 5. Performance

This program is tested with around 10 (200 – 800kb files) which runs the tasks in 1-4 seconds on my local machine (M2 Mac – 8 Gb memory)

### 5.1 Network overhead

This implementation sends files across process using the gRpc framework. Though it uses the protocol buffers as intermediate files. It spent around 50ms to send the files (avg 500kb) over local network.

## 6. Limitations

- Network overhead: all files are sent over network.
- Some read/write errors are not handled.
- Scaling to more mappers and reducers on a single machine is hard. Tested it with 10 mappers and 7 reducers.
- Memory is used to buffer the data, when huge files are read program uses swap memory and performance is affected.
- Fault tolerance is not completely implemented. Though the master checks the heartbeat messages from mappers and reducers using keep alive connections.

## 7. Improvements

- A distributed file system can be used to reduce network bandwidth.
- Can be deployed to cloud functions to make true distributed.
- A good hashing function could improve task load on the reducers.
- Fault tolerance can be implemented.
- Stream connections and connection pooling can be used to reduce the connections overhead.
- Distributed Memcached servers can be used as a shared storage to decrease the network overhead.
- Lot More


## 8. References

[1] Jeffery Dean and Sanjay Ghemawat, Google Inc., Map Reduce: Simplified Data Processing in Large Clusters

[2] Robert Morris, MIT 6.824 Spring 2020, Distributed Systems Course Materials
