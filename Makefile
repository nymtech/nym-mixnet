OUTDIR=build

all:
	make build_client
	make build_mixnode
	make build_provider
	make build_bench_client
	make build_bench_provider

build_client:
	mkdir -p build
	go build -o $(OUTDIR)/loopix_client ./cmd/loopix-client

build_mixnode:
	mkdir -p build
	go build -o $(OUTDIR)/loopix_mixnode ./cmd/loopix-mixnode

build_provider:
	mkdir -p build
	go build -o $(OUTDIR)/loopix_provider ./cmd/loopix-provider

build_bench_client:
	mkdir -p build
	go build -o $(OUTDIR)/bench_loopix_client ./cmd/bench-loopix-client

build_bench_provider:
	mkdir -p build
	go build -o $(OUTDIR)/bench_loopix_provider ./cmd/bench-loopix-provider

