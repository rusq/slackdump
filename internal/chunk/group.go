package chunk

// Catalogue is the generic interface for opening a file with a given version.
type Catalogue interface {
	OpenVersion(FileID, int64) (*File, error)
}

// Group is a group of files with the same FileID and different versions.
type Group struct {
	cat      Catalogue
	versions fileVersions
}
