.PHONY: run
run:
	go run ./cmd/service/main.go -config config.yaml

.PHONY: up
up:
	git clone https://gitlab.com/IgorNikiforov/swordfish-emulator-go.git
	cd swordfish-emulator-go && docker build -t swordfish-emulator . && docker run -d --privileged --name swordfish -p 8080:8080 swordfish-emulator

.PHONY: clean
clean:
	docker stop swordfish
	docker rm swordfish
	rm -rf swordfish-emulator-go
