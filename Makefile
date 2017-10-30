binaries:
	$(MAKE) -C in
	$(MAKE) -C out
	$(MAKE) -C check

docker: binaries
	docker build -t pipelineci/cloudformation-resource:testing .
	docker push pipelineci/cloudformation-resource:testing
