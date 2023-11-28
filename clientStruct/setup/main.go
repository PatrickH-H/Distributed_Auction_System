package main

import (
	"Distributed_Mutual_Exclusion/clientStruct/node"
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("Enter name followed by port number")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()

	input := scanner.Text()
	parts := strings.Fields(input)

	if len(parts) != 2 {
		fmt.Println("Invalid input. Please enter name followed by port number.")
		return
	}

	name := parts[0]
	port := parts[1]
	node := &node.Node{Name: name, Addr: port}
	node.Start()
}
