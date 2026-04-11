run:
	docker-compose up --build

proto:
	protoc --go_out=. --go-grpc_out=. proto/*.proto