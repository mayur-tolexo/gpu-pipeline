BINARY= mq-server
IMAGE= mq:latest
CONFIG= config.json

.PHONY: all build docker test lint helm run clean config

all: build

build:
	go build -o bin/$(BINARY) ./cmd/mq-server

docker: build
	docker build -t $(IMAGE) .

test:
	go test ./... 

helm:
	helm upgrade --install mq chart/mq -f chart/mq/values.yaml

run: build
	./bin/$(BINARY) -config $(CONFIG)

config:
	@echo '{"listen":":8080","partitions":3}' > $(CONFIG) && echo "wrote $(CONFIG)"

clean:
	rm -f bin/$(BINARY) $(CONFIG)
