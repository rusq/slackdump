package source

import "github.com/rusq/slackdump/v3/internal/chunk/dbproc"

type Database struct {
	s *dbproc.Source
}

func OpenDatabase(path string) (*Database, error) {
	s, err := dbproc.Open(path)
	if err != nil {
		return nil, err
	}

	return &Database{s: s}, nil
}
