#!/bin/bash
BUILD_DIR=$(pwd)
ARTIFACT=${ARTIFACT}
mkdir -p /go/src/github.com/ci-pipeline/
cp -r cloudformation-resource /go/src/github.com/ci-pipeline/
cd /go/src/github.com/ci-pipeline/${ARTIFACT}
cp ${ARTIFACT} ${BUILD_DIR}/${ARTIFACT}/${ARTIFACT}
