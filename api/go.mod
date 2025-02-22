module github.com/kubeflow/pipelines/api

go 1.21

require (
	google.golang.org/genproto v0.0.0-20211026145609-4688e4c4e024
	google.golang.org/protobuf v1.27.1
	github.com/kubeflow/pipelines/common v0.0.0
)

replace (
	github.com/kubeflow/pipelines/common => ../common
)
