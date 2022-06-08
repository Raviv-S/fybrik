#!/usr/bin/env bash
# Copyright 2020 The Kubernetes Authors.
# SPDX-License-Identifier: Apache-2.0

source ./common.sh


header_text "Checking for bin/etcd, bin/kube-apiserver and bin/kubectl ${K8S_VERSION}"
[[ -f bin/etcd && -f bin/kubectl && -f bin/kube-apiserver && `bin/kubectl version -o=yaml 2> /dev/null | bin/yq e '.clientVersion.gitVersion' -` == "v${K8S_VERSION}" ]] && exit 0
header_text "Installing bin/etcd, bin/kube-apiserver and bin/kubectl ${K8S_VERSION}"

mkdir -p ./bin
curl -sSLo envtest-bins.tar.gz "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-${K8S_VERSION}-$(go env GOOS)-$(go env GOARCH).tar.gz"

tar -zvxf envtest-bins.tar.gz
mv kubebuilder/bin/* bin
rm envtest-bins.tar.gz
rm -r kubebuilder