build:
	go build -ldflags="-s -w" -trimpath -o loadtest .