.PHONY: api worker gateway migration gen-route gen-repository gen-docs gen-prompt docs install-hooks

api:
	go run ./cmd/api

worker:
	go run ./cmd/worker

gateway:
	go run ./cmd/gateway

migration:
	go run ./cmd/migration

gen-route:
	go run ./cmd/codegen route $(if $(DRY_RUN),--dry-run,) $(if $(CHECK),--check,) $(if $(VERBOSE),--verbose,) $(if $(FORMAT),--format $(FORMAT),)

gen-repository:
	@if [ -z "$(MODEL)" ]; then echo "MODEL is required, e.g. make gen-repository MODEL=Conversation LABEL=会话"; exit 1; fi
	go run ./cmd/codegen repository -model $(MODEL) $(if $(LABEL),-label $(LABEL),) $(if $(SCOPE),-scope $(SCOPE),) $(if $(DRY_RUN),--dry-run,) $(if $(CHECK),--check,) $(if $(VERBOSE),--verbose,) $(if $(FORMAT),--format $(FORMAT),)

gen-docs:
	go run ./cmd/codegen docs $(if $(DRY_RUN),--dry-run,) $(if $(CHECK),--check,) $(if $(VERBOSE),--verbose,) $(if $(FORMAT),--format $(FORMAT),)

gen-prompt:
	go run ./cmd/codegen prompt $(if $(DRY_RUN),--dry-run,) $(if $(CHECK),--check,) $(if $(VERBOSE),--verbose,) $(if $(FORMAT),--format $(FORMAT),)

# docs: 验证 route / repository / OpenAPI 生成产物与源码同步，适合 CI 与 pre-commit
docs:
	go run ./cmd/codegen route --check --format json
	go run ./cmd/codegen repository --list-models --format json
	go run ./cmd/codegen docs --check --format json
	go run ./cmd/codegen prompt --check --format json

# install-hooks: 通过 core.hooksPath 让钩子随仓库生效（推荐）
install-hooks:
	git config core.hooksPath .githooks
	@echo "git hooks 已激活：hooksPath → .githooks"
	@echo "现在每次 push 前都会自动运行本地 CI 校验，避免流水线失败。"
	@echo "跳过方式：GIT_PUSH_VERIFY=false git push"
