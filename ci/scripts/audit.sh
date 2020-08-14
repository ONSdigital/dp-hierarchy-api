#!/bin/bash -eux

export cwd=$(pwd)

pushd $cwd/dp-hierarchy-api
  make audit
popd