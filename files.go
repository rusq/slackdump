package slackdump

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
)

// Files structure is used for downloading files in conversation
type Files struct {
	Files     []slack.File
	ChannelID string
	SD        *SlackDumper
}

// GetFilesFromMessages returns files from the conversation
func (sd *SlackDumper) GetFilesFromMessages(ch *Channel) *Files {
	return &Files{
		Files:     sd.getFilesFromChunk(ch.Messages),
		ChannelID: ch.ID,
		SD:        sd}
}

func (sd *SlackDumper) getFilesFromChunk(m []Message) []slack.File {
	var files []slack.File

	for i := range m {
		if m[i].Files != nil {
			files = append(files, m[i].Files...)
		}
		// include threaded files
		for _, reply := range m[i].ThreadReplies {
			files = append(files, reply.Files...)
		}
	}
	return files
}

// SaveFileTo saves file to the specified directory
func (sd *SlackDumper) SaveFileTo(dir string, f *slack.File) (int64, error) {
	filePath := filepath.Join(dir, Filename(f))
	file, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	if err := sd.client.GetFile(f.URLPrivateDownload, file); err != nil {
		return 0, errors.WithStack(err)
	}

	return int64(f.Size), nil
}

// Filename returns name of the file
func Filename(f *slack.File) string {
	return fmt.Sprintf("%s-%s", f.ID, f.Name)
}

// DumpToDir downloads all file and places then into `dir`.  Return error on error.
func (ff *Files) DumpToDir(dir string) error {
	const parallelDls = 4
	if len(ff.Files) == 0 {
		return nil
	}

	filesC := make(chan *slack.File, 20)
	done := make(chan bool)
	errC := make(chan error, 1)

	go func() {
		errC <- ff.SD.fileDownloader(dir, filesC, done)
	}()

	for f := range ff.Files {
		filesC <- &ff.Files[f]
	}
	// (1) closing filesQ so that fileDownloader to initiate stop
	close(filesC)
	// wait for it to send the done signal
	<-done

	// if there were no errors, this will not execute.
	if err := <-errC; err != nil {
		return err
	}

	return nil
}

func (sd *SlackDumper) fileDownloader(dir string, files <-chan *slack.File, done chan<- bool) error {
	const parallelDls = 4

	var wg sync.WaitGroup
	dlQ := make(chan *slack.File)
	stop := make(chan bool)

	// downloaded contains file ids that already been downloaded,
	// so we don't download the same file twice
	downloaded := make(map[string]bool)

	defer close(done)

	if err := os.Mkdir(dir, 0777); err != nil {
		if !os.IsExist(err) {
			// channels done is closed by defer
			return err
		}
	}

	worker := func(fs <-chan *slack.File) {
	LOOP:
		for {
			select {
			case file := <-fs:
				// download file
				log.Printf("saving %s, size: %d", Filename(file), file.Size)
				n, err := sd.SaveFileTo(dir, file)
				if err != nil {
					log.Printf("error saving %q: %s", Filename(file), err)
				}
				log.Printf("file %s saved: %d bytes written", Filename(file), n)
			case <-stop:
				// (1)
				break LOOP
			}
		}
		// (2)
		wg.Done()
	}

	// create workers
	for i := 0; i < parallelDls; i++ {
		wg.Add(1)
		go worker(dlQ)
	}

	// files queue must be closed on the sender side (see DumpToDir.(1))
	for f := range files {
		_, ok := downloaded[f.ID]
		if ok {
			log.Printf("already seen %s, skipping", Filename(f))
			continue
		}
		dlQ <- f
		downloaded[f.ID] = true
	}

	// closing stop will terminate all workers (1)
	close(stop)
	// workers mark all WorkGroups as done (2)
	wg.Wait()
	// we send the signal to caller that we're done too
	done <- true
	// or close(done)

	//or
	/*
		for {
				// files queue must be closed on the sender side
				f, more := <-files
				if !more {
					close(stop)
					wg.Wait()
					done <- true
					break
				}
				// download file
				dlQ <- f
			}
	*/

	return nil
}
