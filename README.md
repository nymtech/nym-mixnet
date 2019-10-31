# Anonymous messaging using mix networks

This is an implementation of a private communications system based on
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

To initialise a mix client, run:

```shell
./build/loopix-client init --id <YOUR_ID> --local
```

To run the previously initialised client, run:
```shell
./build/loopix-client run --id <YOUR_ID> 
```

If you run the command in two different terminal windows whilst providing different IDs, you will be able to communicate between those clients.