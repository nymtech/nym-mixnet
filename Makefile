OUTDIR=build

all:
	make build_client
	make build_mixnode
	make build_provider

build_client:
	mkdir -p build
	go build -o $(OUTDIR)/loopix_client ./cmd/loopix-client

build_mixnode:
	mkdir -p build
	go build -o $(OUTDIR)/loopix_mixnode ./cmd/loopix-mixnode

build_provider:
	mkdir -p build
	go build -o $(OUTDIR)/loopix_provider ./cmd/loopix-provider

