.PHONY: build image clean run dev

build:
	cd web && npm install && npm run build
	rm -rf internal/web/dist
	cp -r web/dist internal/web/dist
	go build -o huginn ./cmd/huginn

image:
	docker build -f serverBuild/Dockerfile -t hugin-server .

clean:
	-docker stop $$(docker ps -aq)
	-docker rm $$(docker ps -aq)

run:
	@echo "BISMILLAAAAAH"
	./huginn start configs/example.yaml

dev: clean build run
