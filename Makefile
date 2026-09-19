run:
	@echo "Starting auth server..."
	go run ./auth/cmd/server/main.go &
	@echo "Starting game server..."
	go run ./game/cmd/server/main.go &

stop:
	@echo "Stopping services..."
	-lsof -ti:8080 | xargs kill -9 2>/dev/null; true
	-lsof -ti:8081 | xargs kill -9 2>/dev/null; true
	-lsof -ti:5173 | xargs kill -9 2>/dev/null; true
	@echo "All services stopped."