package node

import (
	"Distributed_Auctuion_System/Logger"
	DME "Distributed_Auctuion_System/gRPC_commands"
	"bufio"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

func (node *Node) SendMessage(ctx context.Context, message *DME.Message) (*DME.Response, error) {
	keyword := strings.Split(message.GetMessage(), " ")[0]
	switch strings.ToLower(keyword) {
	case "bid":
		if node.Leader != node.Addr {
			return &DME.Response{Responses: "Not lead"}, nil
		}
		Logger.FileLogger.Println("Participant " + message.GetSender() + " want to bid with the amount " + strings.Split(message.GetMessage(), " ")[1])
		amount, _ := strconv.Atoi(strings.Split(message.GetMessage(), " ")[1])
		currentUserAmount, _ := strconv.Atoi(node.AuctionMap[message.GetSender()])
		AuctionLeadAmoutInt, _ := strconv.Atoi(node.AuctionLeadAmount)
		_, exists := node.AuctionMap[message.GetSender()]
		if exists {
			if currentUserAmount < amount {
				node.AuctionMap[message.GetSender()] = strconv.Itoa(amount)
			}
		} else {
			node.AuctionMap[message.GetSender()] = strings.Split(message.GetMessage(), " ")[1]
		}

		tempInt, _ := strconv.Atoi(node.AuctionMap["LAMPORT"])
		tempInt++
		node.AuctionMap["LAMPORT"] = strconv.Itoa(tempInt)
		if node.AuctionState == "ONGOING" {
			if amount > AuctionLeadAmoutInt {
				node.AuctionLeadAmount = strings.Split(message.GetMessage(), " ")[1]
				node.AuctionLeadUser = message.GetSender()
				Logger.FileLogger.Println("Got the response: <Your bid is the highest!> and got sent AuctionLeadAmount, CurrentHighestBidder, AuctionMap and AuctionState")
				return &DME.Response{Responses: "Your bid is the highest!", CurrentBid: node.AuctionLeadAmount, CurrentHighestBidder: message.GetSender(), AuctionMap: node.AuctionMap, AuctionState: node.AuctionState}, nil
			} else {
				Logger.FileLogger.Println("Got the response: <Ongoing auction and your bid was not the highest. Highest bid right now is <highest bid>> and got sent AuctionLeadAmount, CurrentHighestBidder, AuctionMap and AuctionState")
				return &DME.Response{Responses: "Ongoing auction and your bid was not the highest. Highest bid right now is " + node.AuctionLeadAmount, CurrentBid: node.AuctionLeadAmount, CurrentHighestBidder: node.AuctionLeadUser, AuctionMap: node.AuctionMap, AuctionState: node.AuctionState}, nil
			}
		} else if node.AuctionState == "STALE" {
			node.AuctionState = "ONGOING"
			go node.startAuction()
			node.AuctionLeadAmount = strings.Split(message.GetMessage(), " ")[1]
			node.AuctionLeadUser = message.GetSender()
			Logger.FileLogger.Println("Got the response: <You successfully started an auction!> and got sent AuctionLeadAmount, CurrentHighestBidder, AuctionMap and AuctionState")
			return &DME.Response{Responses: "You successfully started an auction! ", CurrentBid: node.AuctionLeadAmount, CurrentHighestBidder: message.GetSender(), AuctionMap: node.AuctionMap, AuctionState: node.AuctionState}, nil
		}
		Logger.FileLogger.Println("Got the response: <Error, try again>")
		return &DME.Response{Responses: "Error, try again"}, nil

	case "ack":
		Logger.FileLogger.Println("Participant " + message.GetSender() + " wants to know if node " + node.Addr + " is available ")
		Logger.FileLogger.Println("Got the response: <UCK>")
		return &DME.Response{Responses: "UCK"}, nil
	case "result":
		Logger.FileLogger.Println("Participant" + message.GetSender() + " wants to know current state of auction")
		if node.AuctionState == "STALE" {
			Logger.FileLogger.Println("Got the response: <No ongoing auction> and got sent AuctionState")
			return &DME.Response{Responses: "No ongoing auction.", AuctionState: node.AuctionState}, nil
		}
		Logger.FileLogger.Println("Got the response: <ONGOING AUCTION! Current bid: <AuctionLeadAmount>> and got sent AuctionLeadAmount, AuctionMap and AuctionState")
		return &DME.Response{Responses: "ONGOING AUCTION! Current bid: " + node.AuctionLeadAmount, AuctionMap: node.AuctionMap, AuctionState: node.AuctionState}, nil
	case "kingupdate":
		Logger.FileLogger.Println("Participant" + message.GetSender() + " wants to know current lamport time of node")
		Logger.FileLogger.Println("Got sent node.Addr, and current lamport time")
		return &DME.Response{Responses: node.Addr, AuctionState: node.AuctionMap["LAMPORT"]}, nil
	case "youareit":
		Logger.FileLogger.Println("Participant" + message.GetSender() + " wants to let node know it is now the leader")
		node.updateLeadPeer()
		if node.AuctionState == "ONGOING" {
			go node.startAuction()
		}
		return &DME.Response{Responses: "BLAAAAH"}, nil
	default:
		return &DME.Response{Responses: "Unknown command. Try 'BID', 'RESULT' or 'QUIT' "}, nil
	}
}

type IndexError struct {
	message string
}

type Node struct {
	Name  string
	Addr  string
	peers map[string]string
	DME.UnimplementedP2PServiceServer

	AuctionState      string
	AuctionLeadUser   string
	AuctionLeadAmount string
	AuctionMap        map[string]string

	oldAuctionMap  map[string]string
	Leader         string
	OngoingEditing string
}

func (node *Node) Start() {
	if node.Addr == "127.0.0.1:10000" {
		node.Leader = node.Addr
	}
	node.OngoingEditing = "NO"
	node.AuctionState = "STALE"
	node.peers = make(map[string]string)
	node.AuctionMap = make(map[string]string)
	node.AuctionMap["LAMPORT"] = "0"
	node.oldAuctionMap = make(map[string]string)
	node.writeConnectedPeers()
	go node.StartListening()
	go node.requestAccess()
	go node.pingPeersOrKing()
	bl := make(chan bool)
	<-bl
}

func (node *Node) StartListening() {
	Logger.FileLogger.Println("User " + node.Name + " has joined the Auction")
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

func (node *Node) requestAccess() {
	for {
		reader := bufio.NewReader(os.Stdin)
		nodeMessage, _ := reader.ReadString('\n')
		nodeMessage = strings.Trim(nodeMessage, "\r\n")

		switch strings.ToLower(strings.Split(nodeMessage, " ")[0]) {
		case "bid":
			if len(strings.Split(nodeMessage, " ")) < 2 {
				go node.requestAccess()
				fmt.Println("No input after bid")
				return
			}
			if _, err := strconv.Atoi(strings.Split(nodeMessage, " ")[1]); err == nil {
				node.writeToPeer(nodeMessage)
			} else {
				fmt.Println("Could not give bid, second was not variable")
			}
		case "result":
			node.writeToPeer(nodeMessage)
		case "quit":
			os.Exit(1)
		default:
			fmt.Println("Unknown command. Try 'BID', 'RESULT' or 'QUIT'")
		}
	}

}

func (node *Node) writeToPeer(message string) {
	node.getLeadPeer()
	conn, err := grpc.Dial(node.Leader, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Unable to connect to %s: %v", node.Leader, err)
	}
	defer conn.Close()
	p2pClient := DME.NewP2PServiceClient(conn)
	response, err := p2pClient.SendMessage(context.Background(), &DME.Message{Message: message, Sender: node.Addr})
	if response != nil {
		responseLamport, _ := strconv.Atoi(response.AuctionMap["LAMPORT"])
		nodeLamport, _ := strconv.Atoi(node.AuctionMap["LAMPORT"])
		fmt.Println(response.GetResponses())
		if responseLamport > nodeLamport {
			node.AuctionMap = response.AuctionMap
			node.AuctionState = response.GetAuctionState()
			node.AuctionLeadUser = response.GetCurrentHighestBidder()
			node.AuctionLeadAmount = response.GetCurrentBid()
		}
	} else {
		fmt.Println("No response from leader...")
	}
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

func (node *Node) getLeadPeer() {
	var logpath = "../../currentLeader.txt"
	var file, _ = os.Open(logpath)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		node.Leader = line
	}
}
func (node *Node) updateLeadPeer() {
	node.OngoingEditing = "YES"
	Logger.FileLogger.Println("New leader updated to: " + node.Addr)
	var logpath = "../../currentLeader.txt"
	newFile, _ := os.OpenFile(logpath, os.O_WRONLY|os.O_TRUNC, 0644)
	defer newFile.Close()

	writer := newFile
	fmt.Fprintln(writer, node.Addr)
}
func (node *Node) deleteDisconnectedPeers(addrToRemove string) {
	var logpath = "../../connectedNode.txt"
	var file, _ = os.Open(logpath)
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, addrToRemove) {
			lines = append(lines, line)
		}
	}
	newFile, _ := os.Create(logpath)
	defer newFile.Close()

	writer := bufio.NewWriter(newFile)
	for _, line := range lines {
		fmt.Fprintln(writer, line)
	}

	err := writer.Flush()
	if err != nil {
		return
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
func (node *Node) startAuction() {
	Logger.FileLogger.Println("Auction has started")
	time.Sleep(15 * time.Second)
	fmt.Println("AUCTION ENDED!")
	Logger.FileLogger.Println("Auction has ended")
	for i := 0; i < 10; i++ {
		if i > 10 {
			fmt.Println("All bidders left the auction, no winner is chosen")
			break
		}
		var maxKey string
		var maxValue int
		// Initialize with a value lower than the possible values in the map
		maxValue = -1
		for key, valueStr := range node.AuctionMap {
			value, err := strconv.Atoi(valueStr)
			if err != nil {
				fmt.Println("Error converting value:", err)
				continue
			}

			if value > maxValue {
				maxValue = value
				maxKey = key
			}
		}
		fmt.Println("WINNER IS " + maxKey + " WITH A BID OF " + node.AuctionMap[maxKey])
		if isGRPCAddressAvailable(maxKey, 5*time.Second) {
			Logger.FileLogger.Println("Successfully connected to winner " + maxKey)
			fmt.Println("REAL WINNER IS " + maxKey + " WITH A BID OF " + node.AuctionMap[maxKey])
			node.oldAuctionMap = node.AuctionMap

			node.AuctionState = "STALE"
			node.AuctionLeadAmount = "0"
			node.AuctionLeadUser = ""
			node.AuctionMap = make(map[string]string)
			tempInt, _ := strconv.Atoi(node.oldAuctionMap["LAMPORT"])
			tempInt++
			node.AuctionMap["LAMPORT"] = strconv.Itoa(tempInt)
			break
		} else {
			Logger.FileLogger.Println("Failed to connected to winner " + maxKey + " choosing another winner")
			fmt.Println("Failed to connect to winner, choosing backup")
		}
		delete(node.AuctionMap, maxKey)
	}
}

func isGRPCAddressAvailable(address string, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	p2pClient := DME.NewP2PServiceClient(conn)
	_, err = p2pClient.SendMessage(context.Background(), &DME.Message{Message: "ACK", Sender: "127.0.0.1:10000"})
	if err != nil {

		return false
	}
	defer conn.Close()

	//fmt.Printf("Successfully connected to %s\n", address)
	return true
}

func (e *IndexError) Error() string {
	return e.message
}

func (node *Node) pingPeersOrKing() {
	for {
		time.Sleep(12 * time.Second)
		if node.Leader == node.Addr {
			node.getConnectedPeers()
			for name, addr := range node.peers {
				nameAddrString := name + " " + addr

				if isGRPCAddressAvailable(addr, 2*time.Second) {
					continue
				} else {
					node.deleteDisconnectedPeers(nameAddrString)
				}
			}
		} else {
			node.getLeadPeer()
			if isGRPCAddressAvailable(node.Leader, 2*time.Second) {
				continue
			} else {
				node.updateKING()
			}
		}
	}

}
func (node *Node) updateKING() {
	lamportMap := make(map[string]string)
	node.getConnectedPeers()
	Logger.FileLogger.Println("Failed to connect to current leader, choosing new leader...")
	for _, addr := range node.peers {
		if isGRPCAddressAvailable(addr, 2*time.Second) {
			conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Printf("Unable to connect to %s: %v", node.Leader, err)
			}
			defer conn.Close()
			p2pClient := DME.NewP2PServiceClient(conn)
			response, err := p2pClient.SendMessage(context.Background(), &DME.Message{Message: "KINGUPDATE"})
			lamportMap[strings.Trim(response.GetResponses(), "\r\n")] = strings.Trim(response.GetAuctionState(), "\r\n")

		}
	}
	var maxKey string
	var maxValue int

	// Initialize with a value lower than the possible values in the map
	maxValue = -1
	for key, valueStr := range lamportMap {
		value, err := strconv.Atoi(valueStr)
		if err != nil {
			fmt.Println("Error converting value:", err)
			continue
		}

		if value > maxValue {
			maxValue = value
			maxKey = key
		}
	}
	conn, err := grpc.Dial(maxKey, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("Unable to connect to %s: %v", node.Leader, err)
	}
	defer conn.Close()
	p2pClient := DME.NewP2PServiceClient(conn)
	p2pClient.SendMessage(context.Background(), &DME.Message{Message: "YOUAREIT"})
}
