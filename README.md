# MyPBFT

MyPBFT is an implementation of the PBFT (Practical Byzantine Fault Tolerance) algorithm based on blockchain.

## Installation and Usage

### 1. Configure User Ports
MyPBFT uses `userconfig` to define the deployment ports.

### 2. Build the PBFT Node
Run the following command, replacing `<port_number>` with the desired port number:

```sh
go build 
./pbft <port_number>
