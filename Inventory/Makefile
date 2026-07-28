BINARY := techtoss

.PHONY: all run seed clean fmt vet

all: $(BINARY)

$(BINARY): $(wildcard *.go)
	go build -o $(BINARY) .

run: $(BINARY)
	./$(BINARY)

# Run with example data in a throwaway database.
seed: $(BINARY)
	./$(BINARY) -seed -dbfile demo.db

fmt:
	go fmt ./...

vet:
	go vet ./...

clean:
	rm -f $(BINARY) demo.db
