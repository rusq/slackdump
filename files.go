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

// Files structure is used for downloading conversation files.
type Files struct {
	Files     []slack.File
	ChannelID string
}

// ChannelFiles returns files from the conversation.
func (sd *SlackDumper) ChannelFiles(ch *Channel) *Files {
	return &Files{
		Files:     sd.filesFromMessages(ch.Messages),
		ChannelID: ch.ID,
	}
}

// filesFromMessages extracts files from messages slice.
func (sd *SlackDumper) filesFromMessages(m []Message) []slack.File {
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
	filePath := filepath.Join(dir, filename(f))
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

// filename returns name of the file
func filename(f *slack.File) string {
	return fmt.Sprintf("%s-%s", f.ID, f.Name)
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
				log.Printf("saving %s, size: %d", filename(file), file.Size)
				n, err := sd.SaveFileTo(dir, file)
				if err != nil {
					log.Printf("error saving %q: %s", filename(file), err)
				}
				log.Printf("file %s saved: %d bytes written", filename(file), n)
			case <-stop:
				break LOOP
			}
		}
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
			log.Printf("already seen %s, skipping", filename(f))
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

	return nil
}
