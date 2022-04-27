all: resample proxy

resample:
	go build -o resample.so -buildmode=plugin ./pkg/resample/lib

proxy:
	go build -o proxy ./cmd/proxy

clean:
	rm -rf resample.so proxy

.PHONY: clean
