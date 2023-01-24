TAG := $(shell git rev-parse --short HEAD)##TODO: tag by date. we may need to add a build.last file to keep track of last build version number
CUSTOMTAG ?=
BASE_TAG := $(shell cat base.last)

DOCKER_BUILD := docker build \
	--build-arg TAG=${TAG}

DOCKER_BUILDX := docker buildx build

.PHONY: images
images: ## Build storagenode Docker images
	${DOCKER_BUILD} --pull=true -t storjlabs/storagenode:${TAG}${CUSTOMTAG}-amd64 \
		--build-arg=GOARCH=amd64 --build-arg=BASE_TAG=${BASE_TAG}-amd64 \
		-f Dockerfile .
	${DOCKER_BUILD} --pull=true -t storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm32v5 \
    	--build-arg=GOARCH=arm --build-arg=BASE_TAG=${BASE_TAG}-arm32v5 --build-arg=DOCKER_PLATFORM=linux/arm/v5 \
        -f Dockerfile .
	${DOCKER_BUILD} --pull=true -t storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm64v8 \
		--build-arg=GOARCH=arm64 --build-arg=BASE_TAG=${BASE_TAG}-arm64v8 --build-arg=DOCKER_PLATFORM=linux/arm64 \
        -f Dockerfile .

.PHONY: push-images
push-images: ## Push Docker images to Docker Hub
	docker push storjlabs/storagenode:${TAG}${CUSTOMTAG}-amd64 \
	&& docker push storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm32v5 \
	&& docker push storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm64v8 \
	&& for t in ${TAG}${CUSTOMTAG} latest; do \
		docker manifest create storjlabs/storagenode:$$t \
		storjlabs/storagenode:${TAG}${CUSTOMTAG}-amd64 \
		storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm32v5 \
		storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm64v8 \
		&& docker manifest annotate storjlabs/storagenode:$$t storjlabs/storagenode:${TAG}${CUSTOMTAG}-amd64 --os linux --arch amd64 \
		&& docker manifest annotate storjlabs/storagenode:$$t storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm32v5 --os linux --arch arm --variant v5 \
		&& docker manifest annotate storjlabs/storagenode:$$t storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm64v8 --os linux --arch arm64 --variant v8 \
		&& docker manifest push --purge storjlabs/storagenode:$$t \
	; done

.PHONY: storagenode-base-image
storagenode-base-image: ## Build storagenode Docker base image. Requires buildx. This image is expected to be built manually using buildx and QEMU.
	${DOCKER_BUILDX} --pull=true -t storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-amd64 \
		-f Dockerfile.base .
	${DOCKER_BUILDX} --pull=true -t storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm32v5 \
    	--build-arg=GOARCH=arm --build-arg=DOCKER_ARCH=arm32v5 \
        -f Dockerfile.base .
	${DOCKER_BUILDX} --pull=true -t storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm64v8 \
		--build-arg=GOARCH=arm64 --build-arg=DOCKER_ARCH=arm64v8 \
        -f Dockerfile.base .

.PHONY: push-storagenode-base-image
push-storagenode-base-image: ## Push the storagenode base image to dockerhub
	docker push storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-amd64
	docker push storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm32v5
	docker push storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm64v8
	# create, annotate and push manifests for latest-amd64
	docker manifest create storjlabs/storagenode-base:latest-amd64 storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-amd64
	docker manifest annotate storjlabs/storagenode-base:latest-amd64 storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-amd64 --os linux --arch amd64
	docker manifest push --purge storjlabs/storagenode-base:latest-amd64
	# create, annotate and push manifests for latest-arm32v5
	docker manifest create storjlabs/storagenode-base:latest-arm32v5 storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm32v5
	docker manifest annotate storjlabs/storagenode-base:latest-arm32v5 storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm32v5 --os linux --arch arm --variant v5
	docker manifest push --purge storjlabs/storagenode-base:latest-arm32v5
	# create, annotate and push manifests for latest-arm64v8
	docker manifest create storjlabs/storagenode-base:latest-arm64v8 storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm64v8
	docker manifest annotate storjlabs/storagenode-base:latest-arm64v8 storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm64v8 --os linux --arch arm64 --variant v8
	docker manifest push --purge storjlabs/storagenode-base:latest-arm64v8
	# create, annotate and push manifests for main ${TAG}${CUSTOMTAG} tag without arch extension and latest tag
	for t in ${TAG}${CUSTOMTAG} latest; do \
    	docker manifest create storjlabs/storagenode-base:$$t \
    	storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-amd64 \
    	storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm32v5 \
    	storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm64v8 \
    	&& docker manifest annotate storjlabs/storagenode-base:$$t storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-amd64 --os linux --arch amd64 \
    	&& docker manifest annotate storjlabs/storagenode-base:$$t storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm32v5 --os linux --arch arm --variant v5 \
    	&& docker manifest annotate storjlabs/storagenode-base:$$t storjlabs/storagenode-base:${TAG}${CUSTOMTAG}-arm64v8 --os linux --arch arm64 --variant v8 \
    	&& docker manifest push --purge storjlabs/storagenode-base:$$t \
    ; done
