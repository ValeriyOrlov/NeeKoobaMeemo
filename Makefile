run:
	@echo "Starting auth server..."
	go run ./auth/cmd/server/main.go &
	@echo "Starting game server..."
	go run ./game/cmd/server/main.go &