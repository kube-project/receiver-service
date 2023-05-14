# -*- mode: Python -*-

kubectl_cmd = "kubectl"

# verify kubectl command exists
if str(local("command -v " + kubectl_cmd + " || true", quiet = True)) == "":
    fail("Required command '" + kubectl_cmd + "' not found in PATH")

# Use kustomize to build the install yaml files
install = read_file('manifests/receiver.yaml')

# Update the root security group. Tilt requires root access to update the
# running process.
objects = decode_yaml_stream(install)
for o in objects:
    if o.get('kind') == 'Deployment' and o.get('metadata').get('name') == 'receiver':
        o['spec']['template']['spec']['securityContext']['runAsNonRoot'] = False
        break

updated_install = encode_yaml_stream(objects)

# Apply the updated yaml to the cluster.
k8s_yaml(updated_install, allow_duplicates = True)

load('ext://restart_process', 'docker_build_with_restart')

local_resource(
    'receiver-binary',
    "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/receiver ./cmd/root.go",
    deps = [
        "go.mod",
        "go.sum",
        "pkg",
        "models",
        "cmd",
    ],
)

# Build the docker image for our controller. We use a specific Dockerfile
# since tilt can't run on a scratch container.
# `only` here is important, otherwise, the container will get updated
# on _any_ file change. We only want to monitor the binary.
# If debugging is enabled, we switch to a different docker file using
# the delve port.
entrypoint = ['/receiver']
dockerfile = 'tilt.dockerfile'

docker_build_with_restart(
    'ghcr.io/kube-project/receiver-service',
    '.',
    dockerfile = dockerfile,
    entrypoint = entrypoint,
    only=[
      './bin',
    ],
    live_update = [
        sync('./bin/receiver', '/receiver'),
    ],
)

k8s_resource('receiver-deployment', port_forwards='8000:8000')
