TAG  = timberiodev/http_test_server:latest

docker-build:
	docker build --tag ${TAG} .

docker-run:
	docker run --interactive --tty --rm -p 8080:8080 ${TAG}

docker-publish: docker-build
	docker push ${TAG}
