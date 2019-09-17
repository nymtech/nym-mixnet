# Anonymous messaging using mix networks

This is an implementation of an private communications system based on
Ania Piotrowska's PhD research. It implements a 
[Loopix](https://arxiv.org/abs/1703.00536) mixnet as well as the 
[Sphinx](https://cypherpunks.ca/~iang/pubs/Sphinx_Oakland09.pdf) packet format.

## Setup

To build and test the code you need:

* Go 1.12 or later

To build the code:

```shell
make
```

To perform the unit tests run:

```shell
go test ./...
```

Before first fresh run of the system run:

```shell
./scripts/clean.sh
```

This removes all log files, local provider inboxes, and database.

## Usage

To run the network, i.e., mixnodes and providers run:

```shell
./scripts/run_network.sh
```

This spins up 3 mixnodes and 1 provider. You can pass arguments to the script
(e.g. `./scripts/run_network.sh 6` if you want to run 6 mixnodes instead of 3. 

To simulate 2 clients that can message each other, run:

```shell
./scripts/run_client1.sh
```
Then in another terminal:

```shell
./scripts/run_client2.sh
```

You can enter messages in each of the client terminals. Hitting `<enter>` will cause the message to send to the other client. 

Client1 shows only messages being sent and received, so it doesn't scroll as actively and annoyingly. Client2 has a lot more log messages in it: this is not as nice to use from a human perspective, but it shows all the drop traffic, cover traffic, and real messages being sent, so you get a much better feel for what's going on. 
