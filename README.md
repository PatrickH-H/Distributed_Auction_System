# Distributed_Mutual_exclusion
Peer-to-Peer auction system implementation using gRPC where participants can bid and see current auction resulsts
Main learning goal is to implement replication to design a service that is resilient to crashes.

To start the system: "go run main.go" in the /clientStruct/setup folder

Start any number of clients in different processes - it will ask for a name and an IP, type in any name folder by a space and an ip address Example: Christian 127.0.0.1:10000 (Note: the first proccess HAS to be 127.0.0.1:10000, all proccess after this can have whatever IP-address)

Using any of the clients type in "BID <amount>" to bid on current auction and "RESULT" to get the status of the current auction

If you wish to run this multiple times for testing, remember to clear and paste "127.0.0.1:10000" into currentLeader.txt. You can also clear connectedNodes.txt if you wish, this is however not needed.
