all: resample agent

resample:
	go build -o resample.so -buildmode=plugin ./pkg/resample/lib

agent:
	go build -o agent ./cmd/proxy
