server:
	JAEGER_SERVICE_NAME=$@ JAEGER_SAMPLER_TYPE=const JAEGER_SAMPLER_PARAM=1 go run server.go

.PHONY: client

client:
	JAEGER_SERVICE_NAME=$@ JAEGER_SAMPLER_TYPE=const JAEGER_SAMPLER_PARAM=1 go run client/client.go

jaeger:
	docker run --rm -it --name jaeger \
		-e COLLECTOR_ZIPKIN_HTTP_PORT=9411 \
		-p 5775:5775/udp \
		-p 6831:6831/udp \
		-p 6832:6832/udp \
		-p 5778:5778 \
		-p 16686:16686 \
		-p 14268:14268 \
		-p 9411:9411 \
		jaegertracing/all-in-one:1.8
