package redownload

// FileStats contains the file statistics.
type FileStats struct {
	NumFiles uint
	NumBytes uint64
}

func (fs *FileStats) add(other FileStats) {
	fs.NumFiles += other.NumFiles
	fs.NumBytes += other.NumBytes
}
