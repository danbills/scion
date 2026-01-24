#!/bin/bash

container image pull us-central1-docker.pkg.dev/ptone-misc/public-docker/scion-gemini:latest
container image pull us-central1-docker.pkg.dev/ptone-misc/public-docker/scion-claude:latest
container image pull us-central1-docker.pkg.dev/ptone-misc/public-docker/scion-opencode:latest
container image pull us-central1-docker.pkg.dev/ptone-misc/public-docker/scion-codex:latest
container image prune
