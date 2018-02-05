all: build

NAME=dell-provisioner-test
REMOTE_NAME=$(DOCKER_ID_USER)/$(NAME)

.PHONY: build
build: $(NAME)
	docker build -t $(NAME) .

.PHONY: tag
tag: build
	docker tag $(NAME) $(REMOTE_NAME):latest 

.PHONY: push
push: check-docker-hub tag
	docker push $(REMOTE_NAME):latest

.PHONY: clean
clean:
	rm -rf tmp

tmp:
	mkdir -p tmp

$(NAME): tmp
	cp ../bin/dell-provisioner-linux-amd64 tmp/$(NAME)

.PHONY: check-docker-hub
check-docker-hub:
ifndef DOCKER_ID_USER
	$(error ERROR! DOCKER_ID_USER environment variable must be defined)
endif
