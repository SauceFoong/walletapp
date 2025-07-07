run:
	swag init --generalInfo cmd/app/main.go --output docs
	go run cmd/app/main.go