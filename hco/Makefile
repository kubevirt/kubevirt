IMAGE = quay.io/kubevirt/csv_gen:latest

build:
	@docker build -t $(IMAGE) .

run:
	@docker run -it $(IMAGE) $(ARGS)

.PHONY: \
	build \
	run
