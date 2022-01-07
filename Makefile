.DEFAULT_GOAL=image

image:
	docker build -t zadenis/pinger:0.1 .
	docker push zadenis/pinger:0.1
.PHONY: image