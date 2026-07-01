.PHONY: api migration gen-route gen-repository

api:
	go run ./cmd/api

migration:
	go run ./cmd/migration

gen-route:
	go run ./cmd/codegen route $(if $(DRY_RUN),--dry-run,) $(if $(CHECK),--check,) $(if $(VERBOSE),--verbose,) $(if $(FORMAT),--format $(FORMAT),)

gen-repository:
	@if [ -z "$(MODEL)" ]; then echo "MODEL is required, e.g. make gen-repository MODEL=Conversation LABEL=会话"; exit 1; fi
	go run ./cmd/codegen repository -model $(MODEL) $(if $(LABEL),-label $(LABEL),) $(if $(SCOPE),-scope $(SCOPE),) $(if $(DRY_RUN),--dry-run,) $(if $(CHECK),--check,) $(if $(VERBOSE),--verbose,) $(if $(FORMAT),--format $(FORMAT),)
