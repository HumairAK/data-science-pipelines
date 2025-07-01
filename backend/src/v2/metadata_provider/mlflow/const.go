package mlflow

// special tags in MLFlow

const (
	// built ins
	ExperimentDescriptionTag = "mlflow.note.content"
	ParentTag                = "mlflow.parentRunId"
	// kfp meta tags
	NamespaceTag = "kfp.namespace"
	NameTag      = "kfp.name"
)
