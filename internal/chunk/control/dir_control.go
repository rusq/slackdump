// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.
// Package control holds the implementation of the Slack Stream controller.
// It runs the API scraping in several goroutines and manages the data flow
// between them.  It records the output of the API scraper into a chunk
// directory.  It also manages the transformation of the data, if the caller
// is interested in it.
package control

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/trace"

	dirproc "github.com/rusq/slackdump/v4/internal/chunk/backend/directory"

	"github.com/rusq/slackdump/v4/internal/chunk"
	"github.com/rusq/slackdump/v4/internal/structures"
	"github.com/rusq/slackdump/v4/processor"
)

// DirController is the main controller of the Slack Stream.  It runs the API
// scraping in several goroutines and manages the data flow between them.
//
// Deprecated: use [Control] instead.
type DirController struct {
	// chunk directory to store the scraped data.
	cd *chunk.Directory
	// streamer is the API scraper.
	s Streamer
	options
}

// NewDir creates a new [DirController]. Once the [Control.Close] is called it
// closes all file processors.
//
// Deprecated: use [New] instead.
func NewDir(cd *chunk.Directory, s Streamer, opts ...Option) *DirController {
	c := &DirController{
		cd: cd,
		s:  s,
		options: options{
			lg:    slog.Default(),
			tf:    &noopExpTransformer{},
			filer: &noopFiler{},
			avp:   &noopAvatarProc{},
		},
	}
	for _, opt := range opts {
		opt(&c.options)
	}
	return c
}

func (c *DirController) Run(ctx context.Context, list *structures.EntityList) error {
	ctx, task := trace.NewTask(ctx, "Controller.Run")
	defer task.End()

	var dcp processor.Channels = &processor.NopChannels{}
	if !list.HasIncludes() { // all channels are included
		if p, err := dirproc.NewChannels(c.cd); err != nil {
			return Error{"channel", "init", err}
		} else {
			dcp = p
		}
	}
	dwsp, err := dirproc.NewWorkspace(c.cd)
	if err != nil {
		return Error{"workspace", "init", err}
	}
	dconv, err := dirproc.NewConversation(c.cd, c.filer, c.tf, dirproc.WithRecordFiles(c.flags.RecordFiles))
	if err != nil {
		return fmt.Errorf("error initialising conversation processor: %w", err)
	}
	dusr, err := dirproc.NewUsers(c.cd)
	if err != nil {
		return Error{"user", "init", err}
	}
	userproc := processor.JoinUsers(
		c.newUserCollector(ctx, false), // chunk directory does not support channel users.
		dusr,
		c.avp,
	)

	mp := superprocessor{
		Channels:      dcp,
		WorkspaceInfo: dwsp,
		Users:         userproc,
		Conversations: dconv,
	}

	return runWorkers(ctx, c.s, list, mp, c.flags)
}

// Close closes the controller and all its file processors.
func (c *DirController) Close() error {
	var errs error
	if c.avp != nil {
		if err := c.avp.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("error closing avatar processor: %w", err))
		}
	}
	if c.filer != nil {
		if err := c.filer.Close(); err != nil {
			errs = errors.Join(errs, fmt.Errorf("error closing file processor: %w", err))
		}
	}
	return errs
}
