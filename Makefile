OUTDIR=build

all:
	make build_client
	make build_mixnode
	make build_provider
	make build_bench_client
	make build_bench_provider

build_client:
	mkdir -p build
	go build -o $(OUTDIR)/nym-mixnet-client ./cmd/nym-mixnet-client

build_mixnode:
	mkdir -p build
	go build -o $(OUTDIR)/nym-mixnode ./cmd/nym-mixnode

build_provider:
	mkdir -p build
	go build -o $(OUTDIR)/nym-mixnet-provider ./cmd/nym-mixnet-provider

build_bench_client:
	mkdir -p build
	go build -o $(OUTDIR)/bench-nym-mixnet-client ./cmd/bench-nym-mixnet-client

build_bench_provider:
	mkdir -p build
	go build -o $(OUTDIR)/bench-nym-mixnet-provider ./cmd/bench-nym-mixnet-provider

