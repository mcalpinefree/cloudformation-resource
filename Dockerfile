FROM golang:1.7-wheezy

ADD ./ /go/src/github.com/ci-pipeline/cloudformation-resource

RUN mkdir -p /opt/resource

RUN cd /go/src/github.com/ci-pipeline/cloudformation-resource/check \
	&& go get \
	&& go build \
	&& mv check /opt/resource/check

RUN cd /go/src/github.com/ci-pipeline/cloudformation-resource/out \
	&& go get \
	&& go build \
	&& mv out /opt/resource/out

RUN rm -rf /go/src/github.com/ci-pipeline/cloudformation-resource
