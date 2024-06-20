package types

import (
	"io"
)

// ArtifactDriver is the interface for loading and saving of artifacts
type ArtifactDriver interface {
	// OpenStream opens an artifact for reading. If the artifact is a file,
	// then the file should be opened. If the artifact is a directory, the
	// driver may return that as a tarball. OpenStream is intended to be efficient,
	// so implementations should minimise usage of disk, CPU and memory.
	// Implementations must not implement retry mechanisms. This will be handled by
	// the client, so would result in O(nm) cost.
	OpenStream(a *Artifact) (io.ReadCloser, error)
}
