package node

import (
	"Distributed_Mutual_Exclusion/Logger"
	DME "Distributed_Mutual_Exclusion/gRPC_commands"
	"bufio"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

type Node struct {
	Name             string
	Addr             string
	CurrentState     string
	CurrentStateList []int
	peers            map[string]string
	DME.UnimplementedP2PServiceServer
}

func (node *Node) EnterCriticalSection() {
	log.Printf("ENTERED CRITICAL SECTION")
	Logger.FileLogger.Println(node.Name + " entered the critical section!")
	time.Sleep(10 * time.Second)
	log.Printf("LEFT CRITICAL SECTION")
	Logger.FileLogger.Println(node.Name + " left the critical section!")
}

func (node *Node) SendMessage(ctx context.Context, message *DME.Message) (*DME.Response, error) {
	if message.GetMessage() == "LET ME IN" {
		Logger.FileLogger.Println("Node wants access to critical section. " + node.Name + " answering with state: " + node.CurrentState)
		return &DME.Response{Responses: node.CurrentState}, nil
	} else if message.GetMessage() == "REPLY" && node.CurrentState == "WANTED" {
		Logger.FileLogger.Println(node.Name + " got 'REPLY' message")
		node.CurrentStateList = append(node.CurrentStateList, 1)
		return &DME.Response{Responses: node.Name + " " + node.CurrentState}, nil
	} else {
		return &DME.Response{Responses: node.CurrentState}, nil
	}
}

func (node *Node) StartListening() {
	Logger.FileLogger.Println("User " + node.Name + " started listening")
	lis, err := net.Listen("tcp", node.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	DME.RegisterP2PServiceServer(grpcServer, node)
	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (node *Node) Start() {
	node.peers = make(map[string]string)
	node.CurrentState = "RELEASED"
	node.writeConnectedPeers()
	node.getConnectedPeers()
	go node.StartListening()

	go node.requestAccess()
	bl := make(chan bool)
	<-bl
}

func (node *Node) requestAccess() {
	for {
		reader := bufio.NewReader(os.Stdin)
		nodeMessage, _ := reader.ReadString('\n')
		nodeMessage = strings.Trim(nodeMessage, "\r\n")
		if nodeMessage == "LET ME IN" {
			node.getConnectedPeers()
			Logger.FileLogger.Println(node.Name + " wants to enter critical section, waiting for respons...")
			node.CurrentState = "WANTED"
			err := node.getPeerStates()
			if err != nil {
				return
			}
		}
	}
}

func (node *Node) getPeerStates() error {
	//get response of peers
	node.CurrentStateList = make([]int, 0)
	for name, addr := range node.peers {
		if name != node.Name {
			var stateResponse = node.connectToPeer(name, addr)
			if stateResponse == "RELEASED" {
				node.CurrentStateList = append(node.CurrentStateList, 1)
			}
		}
	}

	//for-loop to keep checking if all peers have responded
	for {
		if len(node.CurrentStateList) == len(node.peers)-1 {
			Logger.FileLogger.Println(node.Name + " got respons from all nodes and is now heading to critical section...")
			node.EnterCriticalSection()
			for name, addr := range node.peers {
				if name != node.Name {
					conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
					if err != nil {
						log.Printf("Unable to connect to %s: %v", addr, err)
					}
					defer conn.Close()
					p2pClient := DME.NewP2PServiceClient(conn)
					p2pClient.SendMessage(context.Background(), &DME.Message{Message: "REPLY"})
				}
			}
			node.CurrentState = "RELEASED"
			break
		}
	}
	return nil
}
func (node *Node) connectToPeer(name string, addr string) (stateResponse string) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Unable to connect to %s: %v", addr, err)
		return
	}
	defer conn.Close()
	p2pClient := DME.NewP2PServiceClient(conn)
	response, err := p2pClient.SendMessage(context.Background(), &DME.Message{Message: "LET ME IN"})
	if err != nil {
		log.Printf("Error making request to %s: %v", name, err)
		return
	}
	log.Printf("Got the respons: %s", response.GetResponses())
	return response.GetResponses()
}

func (node *Node) getConnectedPeers() {
	var logpath = "../../connectedNode.txt"
	var file, _ = os.Open(logpath)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		node.peers[parts[0]] = parts[1]
	}
}

func (node *Node) writeConnectedPeers() {
	var logpath = "../../connectedNode.txt"
	var file, _ = os.OpenFile(logpath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	data := []byte(node.Name + " " + node.Addr + "\n")
	_, err := file.Write(data)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

}
