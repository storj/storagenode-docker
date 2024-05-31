TAG := $(shell git rev-parse --short HEAD)##TODO: tag by date. we may need to add a build.last file to keep track of last build version number
CUSTOMTAG ?=

DOCKER_BUILDX := docker buildx build

.PHONY: images
images: amd64-image arm64-image arm32-image ## Build storagenode Docker images

.PHONY: amd64-image
amd64-image: ## Build storagenode Docker image for amd64
	${DOCKER_BUILDX} --load --pull=true -t storjlabs/storagenode:${TAG}${CUSTOMTAG}-amd64 \
		--platform=linux/amd64 \
		--build-arg=GOARCH=amd64 \
		-f Dockerfile .

.PHONY: arm32-image
arm32-image: ## Build storagenode Docker image for arm32v5
	${DOCKER_BUILDX} --load --pull=true -t storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm32v5 \
		--platform=linux/arm/v5 \
		--build-arg=GOARCH=arm --build-arg=DOCKER_ARCH=arm32v5 --build-arg=DOCKER_PLATFORM=linux/arm/v5 \
		-f Dockerfile .

.PHONY: arm64-image
arm64-image: ## Build storagenode Docker image for arm64v8
	${DOCKER_BUILDX} --load --pull=true -t storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm64v8 \
		--platform=linux/arm64/v8 \
		--build-arg=GOARCH=arm64 --build-arg=DOCKER_ARCH=arm64v8 --build-arg=DOCKER_PLATFORM=linux/arm64 \
		-f Dockerfile .

.PHONY: pull-images
pull-images:
	docker pull storjlabs/storagenode:${TAG}${CUSTOMTAG}-amd64
	docker pull storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm32v5
	docker pull storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm64v8

.PHONY: push-images
push-images:
#	$(MAKE) push-images-to-repo REPO=storjlabs
	docker tag storjlabs/storagenode:${TAG}${CUSTOMTAG}-amd64 ghcr.io/storj/storagenode:${TAG}${CUSTOMTAG}-amd64
	docker tag storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm32v5 ghcr.io/storj/storagenode:${TAG}${CUSTOMTAG}-arm32v5
	docker tag storjlabs/storagenode:${TAG}${CUSTOMTAG}-arm64v8 ghcr.io/storj/storagenode:${TAG}${CUSTOMTAG}-arm64v8
	$(MAKE) push-images-to-repo REPO=ghcr.io/storj

.PHONY: push-images-to-repo
push-images-to-repo: ## Push Docker images
	docker push ${REPO}/storagenode:${TAG}${CUSTOMTAG}-amd64 \
	&& docker push ${REPO}/storagenode:${TAG}${CUSTOMTAG}-arm32v5 \
	&& docker push ${REPO}/storagenode:${TAG}${CUSTOMTAG}-arm64v8 \
	&& for t in ${TAG}${CUSTOMTAG} latest; do \
		docker manifest create ${REPO}/storagenode:$$t \
		${REPO}/storagenode:${TAG}${CUSTOMTAG}-amd64 \
		${REPO}/storagenode:${TAG}${CUSTOMTAG}-arm32v5 \
		${REPO}/storagenode:${TAG}${CUSTOMTAG}-arm64v8 \
		&& docker manifest annotate ${REPO}/storagenode:$$t ${REPO}/storagenode:${TAG}${CUSTOMTAG}-amd64 --os linux --arch amd64 \
		&& docker manifest annotate ${REPO}/storagenode:$$t ${REPO}/storagenode:${TAG}${CUSTOMTAG}-arm32v5 --os linux --arch arm --variant v5 \
		&& docker manifest annotate ${REPO}/storagenode:$$t ${REPO}/storagenode:${TAG}${CUSTOMTAG}-arm64v8 --os linux --arch arm64 --variant v8 \
		&& docker manifest push --purge ${REPO}/storagenode:$$t \
	; done
