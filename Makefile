.PHONY=image

forerunner:
	GOOS=linux go build

image: forerunner
	docker build -t brianhicks/forerunner:$$(cat VERSION) .

run: image
	docker run -i -t --rm brianhicks/forerunner:$$(cat VERSION) $(ARGS)
