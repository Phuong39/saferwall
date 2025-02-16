# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 11/11/2022

### Added

- Sandbox agent health check + basic sysinfo and env data collection [##395](https://github.com/saferwall/saferwall/pull/395).
- Push sandbox payload results to the aggregator [#391](https://github.com/saferwall/saferwall/pull/391).
- MultiAV McAfee enable scan for potentially unwanted program [#387](https://github.com/saferwall/saferwall/pull/387).
- Numerous updates to support different types of messages for the aggregator [#383](https://github.com/saferwall/saferwall/pull/383).
    - Add methods for the `storage` internal pkg to support bucket creation.
    - Generate thumbnails for the sandbox screenshots and add health checks for VMs.
    - Remove `cluster-autoscaler` form helm chart.
    - Add documentation with the communication format used between services.
- Agent: collect screenshots and memdumps [#380](https://github.com/saferwall/saferwall/pull/380).
- Guess file extension and include PE signature [#379](https://github.com/saferwall/saferwall/pull/379).
- Curate PE scan results [#378](https://github.com/saferwall/saferwall/pull/378).
- Add `inlets-operator` and `metallb` charts [#376](https://github.com/saferwall/saferwall/pull/376). `inlets-operator` has been deleted later, and `metallb` is installed separately from the chart dependencies.
- Add `kube-prometheus-stack` CRDs and experiment with k3s for local dev.
- Add `workflow_dispatch` for `helm-release` and `release` services job.

### Changed

- Set k8s version to the same as prod k8s version and update default user/password values in minio helm chart [#392](https://github.com/saferwall/saferwall/pull/392).
- Change protobuf message scheme to support uploading object to s3 [#383](https://github.com/saferwall/saferwall/pull/383).
- Bind k8s port forwarding services to `0.0.0.0`.
- Bump wait-for and golang docker images.
- Bump `yara`, `helm`, `kuberneters`, `exiftool`, `kind`, `kubens/kubectx` and `kube-capacity`.
- Bump `aws-efs-csi-driver`, `ingress-nginx`, `couchbase-operator` and `minio` helm chart dependencies.

### Fixed

- Use wine + loadlibrary to make windows defender works again thanks to [prsyahmi](https://github.com/prsyahmi) [#386](https://github.com/saferwall/saferwall/pull/386).
- MultiAV McAfee doesn't report other kind of malware besides trojan thanks to [prsyahmi](https://github.com/prsyahmi) [#387](https://github.com/saferwall/saferwall/pull/387).
- Do not set the file extension/format when it is now known [#381](https://github.com/saferwall/saferwall/pull/381).
- MultiAV upgrade Avast to a newer major release.

## [0.3.0] - 12/04/2022

### Added

- Add pre-commit-config.yaml.
- Update packer/installer/protector sigs and file magic data.
- Introduce new env variables in the UI k8s manifests.
- Add antivirus detections to the list of tags.
- Cleanup file that has not been accessed since a day from the nfs share.
- Documenting saferwall architecture.
- Saferwall sandbox microservice.

### Changed

- Change minio operator to the basic minio.
- Move private go packages to `internal/` directory.
- Move helm chart from its own repo to main repo.
- Numerous tolling updates: docker-compose, devContainers, and bumping go pkg dependencies.

### Fixed

- Fix crash on webapis k8s manifest when generating the toml config.

## [0.2.0] - 25/11/2021

### Added

- Unit tests for ASCII & Unicode strings and AV label pkg.
- [exiftool] ELF binary testcases.
- [yara]: implement yara scanner and update go package version.
- [kubernetes] AWS spot instance template.
- Introduce a new package for virt-manager.
### Fixed

- [magic] Handle case where input is empty.
- [magic] fix out of bounds errors due to file help output on null input.

### Changed

- Move cli to a separate github repository
- Clean up package tests + add tests for `HashBytes` func.
- Update crypto functions to follow idiomatic initialisms.
-[bytestats]  remove python3 poc + use package fixtures for testing.
- Using `zap` instead of `logrus` and asbtract the logging code.
- Asbtract access to object storage and to the database.
- Move the multiav package to a separate repo.
- Separate the consumer into different services (orchestrator, aggregator, pe, metadata, multiav, ML, post-processor).
- Use external NSQ helm chart.

## [0.1.0] - 30/04/2021

### Added

- ML PE classifier and string ranker.
- docker-compose and .devcontainer to ease development.
- A portable executable (PE) file parser.
- A UI for displaying PE parsing results.
- `gib`: a package to detect gibberish strings.
- `bytestats`: a package that implements byte and entropy statistics for binary files.
- `cli` utility to interact with saferwall web apis.
- `sdk2json`: a package to convert Win32 API definitions to JSON format.

### Changed

- Consumer docker image is separated to a base image and an app image.
- Refactor consumer and make it a go module.
- [Helm] reduce minio MEM request, ES and Kibana CPU request to half a core.
- [Helm] bump chart dependency modules.
- [pkg/consumer] add context timeout to multiav scan gRPC API.
- Move the website, the dashboard and the web apis projects to a separate git repos.
- Improvement in CI/CD pipeline: include code coverage, test only changed modules & running custom github action runners.

## [0.0.3] - 2021-15-01

### Added

- A new antivirus engine (DrWeb).
- A new antivirus engine (TrendMicro).
- A Vagrant image (virtualbox) to test locally the product.

### Changed

- Add config option to choose log level.
- Add various labels to k8s manifests and enforce resource req and limits.
- Create seconday indexes for couchbase n1ql queries.
- Replaced CircleCI with Github actions for unit testing go packages.
- Force fail multiav docker build if eicar scanning fails.
- Display only enabled antivirus thanks to [@nikAizuddin](https://github.com/nikAizuddin): [#248](https://github.com/saferwall/saferwall/pull/248)
- Use specific Kubectl version.
- Remove none driver support for `minikube` and replace it with `kind`.
- Bump cert-manager, EKF, Prometheus, ingress-nginx, minio and efs-provionner, couchbase helm chart versions.
- Retry building UI/Backend/MultiAV/Consumer docker imgs one more time when failed.
- Improve the CONTRIBUTING doc.

# Fixed:

- Force lower case a sha256 hash before search in Backend thanks to [@hotail](https://github.com/hotail)
- `AV_LIST` variable in multiav mk was override somewhere thanks to [@najashark](https://github.com/najashark)
- Remove `add_kubernetes_metadata` from filebeat config which was causing duplicated data to be sent to kibana and ddosing the kube api server.

## [0.0.2] - 2020-08-12

### Added

- Add a cmd tool to batch upload files.
- Add s3upload pkg to simplify mass-uploading of files into s3.
- Add upload pkg to simplify uploading a local database of samples to saferwall.
- Add Kibana / ElasticSearch / FileBeat helm chart.
- Add Prometheus Operator helm chart.

### Changed

- Add nfs-server-provisionner for local testing in minikube.
- Improve the building process documentation thanks to [Jameel Haffejee](https://github.com/RC114).
- Reworked to file tags schema.
- Improve the rendering of the landing page.
- Fix phrasing in README from [@bf](https://github.com/bf).
- Fix recover from panic routine in parse-pe in consumer.
- Add exception catching in strings pkg.
- Add ContextLogger in consumer to always log sha256.

## [0.0.1] - 2020-03-09

### Added

- Initiale release includes a multi-av scanner + strings + file metadata.
- UI with options to download, rescan, like a sample and share comments.
- User profile to track submissions, followers and see activities.
