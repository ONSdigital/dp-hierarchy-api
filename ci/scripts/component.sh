#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-hierarchy-api
  make test-component
popd
