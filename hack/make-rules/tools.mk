SKIP_INSTALL_CHECK ?= true

define post-install-check
	$(SKIP_INSTALL_CHECK) || go mod tidy
	$(SKIP_INSTALL_CHECK) || git diff --exit-code -- go.mod
endef

some_file:
	echo "This line will only print once"
	touch some_file

INSTALL_TOOLS += yq
yq:
	cd $(TOOLS_DIR); ./install_yq.sh
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/controller-gen
$(TOOLBIN)/controller-gen:
	GOBIN=$(ABSTOOLBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.7.0
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/dlv
$(TOOLBIN)/dlv:
	GOBIN=$(ABSTOOLBIN) go install github.com/go-delve/delve/cmd/dlv@v1.4.1
	$(call post-install-check)

INSTALL_TOOLS += helm
helm:
	cd $(TOOLS_DIR); ./install_helm.sh
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/golangci-lint
$(TOOLBIN)/golangci-lint:
	GOBIN=$(ABSTOOLBIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.1
	$(call post-install-check)

INSTALL_TOOLS += kubebuilder
kubebuilder:
	cd $(TOOLS_DIR); ./install_kubebuilder.sh
	$(call post-install-check)

INSTALL_TOOLS += kustomize
kustomize:
	cd $(TOOLS_DIR); ./install_kustomize.sh
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/kind
$(TOOLBIN)/kind:
	GOBIN=$(ABSTOOLBIN) go install sigs.k8s.io/kind@v0.11.1
	$(call post-install-check)

INSTALL_TOOLS += istioctl
istioctl:
	cd $(TOOLS_DIR); ./install_istio.sh
	$(call post-install-check)

INSTALL_TOOLS += protoc
protoc:
	cd $(TOOLS_DIR); ./install_protoc.sh
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/protoc-gen-doc
$(TOOLBIN)/protoc-gen-doc:
	GOBIN=$(ABSTOOLBIN) go install github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@v1.4.1
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/protoc-gen-go
$(TOOLBIN)/protoc-gen-go:
	go get -d google.golang.org/protobuf/cmd/protoc-gen-go@v1.27
	GOBIN=$(ABSTOOLBIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/protoc-gen-go-grpc
$(TOOLBIN)/protoc-gen-go-grpc:
	GOBIN=$(ABSTOOLBIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/protoc-gen-lint
$(TOOLBIN)/protoc-gen-lint:
	GOBIN=$(ABSTOOLBIN) go install github.com/ckaznocha/protoc-gen-lint@v0.2.1
	$(call post-install-check)

# INSTALL_TOOLS += oc
oc:
	cd $(TOOLS_DIR); ./install_oc.sh
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/misspell
$(TOOLBIN)/misspell:
	GOBIN=$(ABSTOOLBIN) go install github.com/client9/misspell/cmd/misspell@v0.3.4
	$(call post-install-check)

$(TOOLBIN)/license_finder:
	gem install license_finder -v 6.5.0 --bindir=$(ABSTOOLBIN)
	$(call post-install-check)

INSTALL_TOOLS += opa
opa:
	cd $(TOOLS_DIR); ./install_opa.sh
	$(call post-install-check)

INSTALL_TOOLS += fzn-or-tools
fzn-or-tools:
	cd $(TOOLS_DIR); ./install_or_tools.sh
	$(call post-install-check)

INSTALL_TOOLS += vault
vault:
	cd $(TOOLS_DIR); ./install_vault.sh
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/oapi-codegen
$(TOOLBIN)/oapi-codegen:
	GOBIN=$(ABSTOOLBIN) go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.4.2
	$(call post-install-check)

INSTALL_TOOLS += openapi-generator-cli
openapi-generator-cli:
	cd $(TOOLS_DIR); chmod +x ./install_openapi-generator-cli.sh; ./install_openapi-generator-cli.sh
	$(call post-install-check)

INSTALL_TOOLS += $(TOOLBIN)/crdoc
$(TOOLBIN)/crdoc:
	GOBIN=$(ABSTOOLBIN) go install fybrik.io/crdoc@v0.6.1
	$(call post-install-check)

INSTALL_TOOLS += json-schema-generator
json-schema-generator:
	cd $(TOOLS_DIR); ./install_json-schema-generator.sh
	$(call post-install-check)

.PHONY: install-tools
install-tools: $(INSTALL_TOOLS)
	go mod tidy

.PHONY: uninstall-tools
uninstall-tools:
	find $(TOOLBIN) -mindepth 1 -delete
