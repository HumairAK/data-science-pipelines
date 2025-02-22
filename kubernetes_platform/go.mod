module github.com/kubeflow/pipelines/kubernetes_platform

go 1.21

require (
	google.golang.org/protobuf v1.27.1
	github.com/kubeflow/pipelines/common v0.0.0
)

require golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect

replace (
	github.com/kubeflow/pipelines/api => ../api
	github.com/kubeflow/pipelines/common => ../common
)