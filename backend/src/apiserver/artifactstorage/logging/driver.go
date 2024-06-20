package logging

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/artifactstorage/types"
	log "github.com/sirupsen/logrus"
	"io"
	"time"
)

// driver adds a logging interceptor to help diagnose issues with artifacts
type driver struct {
	types.ArtifactDriver
}

func New(d types.ArtifactDriver) types.ArtifactDriver {
	return &driver{d}
}

func (d driver) OpenStream(a *types.Artifact) (io.ReadCloser, error) {
	t := time.Now()
	key, _ := a.GetKey()
	rc, err := d.ArtifactDriver.OpenStream(a)
	log.WithField("artifactName", a.Name).
		WithField("key", key).
		WithField("duration", time.Since(t)).
		WithError(err).
		Info("Stream artifact")
	return rc, err
}
