# Sidecars buildpack [![Build Status](https://travis-ci.com/orange-cloudfoundry/sidecars-buildpack.svg?branch=master)](https://travis-ci.com/orange-cloudfoundry/sidecars-buildpack)

Sidecars is a special buildpack to let you run any processes as sidecar in your application.

Under the hood it wrap [cloud-sidecars](https://github.com/orange-cloudfoundry/cloud-sidecars) cli.

Sidecar can be run beside your app or in front of your app as a reverse proxy to your app.

This buildpack can't be used as a final buildpack and support stacks:
- cflinuxfs2
- cflinuxfs3
- windows2012R2
- windows2016
- windows

**Tip**: Download [cloud-sidecars](https://github.com/orange-cloudfoundry/cloud-sidecars) command line to have a better usage experience

### Buildpack User Documentation

1. Add this the buildpack as the first buildpack on your app manifest 
2. Change the start command to `cloud-sidecars launch`
3. Add a `sidecars-config.yml` and set your config inside (see [configuration doc](#configuration))

**Manifest example**:

```yaml
applications:
  - name: my-app
    buildpacks:
      - sidecars_buildpack
      - staticfile_buildpack
    disk_quota: 1G
    command: cloud-sidecars launch # tips: you can use all cli params from cloud-sidecars, add flag `--log-level debug` to enable debug mode for example
```

**Tips**: You can override start command for your app by creating a file named `Procfile` and add a `start` entry, e.g.:

```yaml
start: start-command-for-app
```


### Configuration

You can see full configuration on [cloud-sidecars](https://github.com/orange-cloudfoundry/cloud-sidecars) doc.

Learn by example with [gobis-server](https://github.com/orange-cloudfoundry/gobis-server) 
as a reverse proxy and [coredns](https://github.com/coredns/coredns) as a beside app:

```yaml
sidecars:
  # Name must be defined for your sidecar
- name: gobis-server
  # Path to execute your sidecar (You can run binary set in PATH)
  # If artifact_url is set, executable path is prefixed directly with download path by cloud-sidecars
  executable: gobis-server
  # This can be empty, it let you download an artifact. Artifacts are unzipped and placed at <dir>/.sidecars/<sidecar name>
  # executable path is prefixed directly with this path by cloud-sidecars
  # work dir for after_download is this directory: <dir>/.sidecars/<sidecar name>
  # It uses https://github.com/ArthurHlt/zipper for downloading artifacts this let you download git, zip, tar, tgz or any other file (they all be uncompressed)
  artifact_uri: https://github.com/orange-cloudfoundry/gobis-server/releases/download/v1.7.0/gobis-server_linux_amd64.zip
  # force type detection for https://github.com/ArthurHlt/zipper
  artifact_type: http
  # Sha1 to ensure to have correct downloaded artifact
  # This is specific sha1 made by zipper, use cloud-sidecars sha1 command to have sha1 to insert here
  # leave empty to not do sha1 check
  artifact_sha1: "94aba65bd7d2dca6fb115716fee3a575fb03ad1d"
  # Run script after downloading your artifact
  # here it renames gobis-server_linux_amd64 to gobis-server
  after_install: "mv * gobis-server"
  # pass args to executable
  args: 
  - "--sidecar"
  - "--sidecar-app-port"
  # this sidecar is defines as reverse proxy, it give a PROXY_APP_PORT env var
  # as bellow you can give args in posix style from env var
  - "${PROXY_APP_PORT}"
  # Set env var for sidecar
  # you can give a value in posix style from env var
  env:
    FOO: "${PATH}"
    KEY: "val"
  # Set env var for app, all app_env found in sidecars will be merged in one
  # you can give a value in posix style from env var
  app_env: {}
  # You can pass a profile file which will be source before executing app
  profiled: ""
  # Set working directory, by defaul it is the dir defined by cli flag --dir
  work_dir: ""
  # Do not put prefix in stdout/stderr for this sidecar
  no_log_prefix: false
  # If true this will override listen port for app and set an PROXY_APP_PORT env var for sidecar
  # If you have multiple sidecar of type reverse proxy it will chain in the order set here.
  is_rproxy: true
  # If true when your sidecar stop it will not stop main app and others sidecars
  no_interrupt_when_stop: false
- name: coredns
  executable: coredns
  artifact_uri: https://github.com/coredns/coredns/releases/download/v1.4.0/coredns_1.4.0_linux_amd64.tgz
  artifact_type: http
  artifact_sha1: 7b3889d26bd9b6cf6687cac8a7358132af24e287
  args:
  - "-dns.port"
  - "1053"
  env: {}
  app_env: {}
  profiled: ""
  work_dir: ""
  no_log_prefix: false
  is_rproxy: false
  no_interrupt_when_stop: false
```


### Building the Buildpack
To build this buildpack, run the following command from the buildpack's directory:

1. Source the .envrc file in the buildpack directory.
```bash
source .envrc
```
To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Install buildpack-packager
```bash
./scripts/install_tools.sh
```

1. Build the buildpack
```bash
buildpack-packager build
```

1. Use in Cloud Foundry
Upload the buildpack to your Cloud Foundry and optionally specify it by name

```bash
cf create-buildpack [BUILDPACK_NAME] [BUILDPACK_ZIP_FILE_PATH] 1
cf push my_app [-b BUILDPACK_NAME]
```

### Testing
Buildpacks use the [Cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass) framework for running integration tests.

To test this buildpack, run the following command from the buildpack's directory:

1. Source the .envrc file in the buildpack directory.

```bash
source .envrc
```
To simplify the process in the future, install [direnv](https://direnv.net/) which will automatically source .envrc when you change directories.

1. Run unit tests

```bash
./scripts/unit.sh
```

1. Run integration tests

```bash
./scripts/integration.sh
```

More information can be found on Github [cutlass](https://github.com/cloudfoundry/libbuildpack/cutlass).

### Reporting Issues
Open an issue on this project

## Disclaimer
This buildpack is experimental and not yet intended for production use.
