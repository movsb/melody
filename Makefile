.PHONY: build
build:
	(cd server && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../docker/melody)

.PHONY: build-and-push-image
build-and-push-image: build
	docker build -t harbor.home.twofei.com/melody docker/
	docker push harbor.home.twofei.com/melody
