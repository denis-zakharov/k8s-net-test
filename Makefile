.PHONY: image

image:
	docker build -t zadenis/pinger:0.1 pinger
	docker push zadenis/pinger:0.1