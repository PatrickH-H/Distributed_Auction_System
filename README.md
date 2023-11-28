# Distributed_Mutual_exclusion
Peer-to-Peer implementation using gRPC where Peers can enter a Critical Section

Start a peer with: "go run main.go" in the /clientStruct/setup folder

Start any number of clients in different processes - i twill ask for a name and an IP, type in any name folder by a space and an ip address
Example:
Christian 127.0.0.1:10000
(Note: I used 127.0.0.1:1xxxx for all my testing)

Using any of the clients type in "LET ME IN" to get access to the critical section.
