package convert

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/rusq/slackdump/v3/source/mock_source"

	"github.com/rusq/fsadapter"
	"github.com/rusq/fsadapter/mocks/mock_fsadapter"
	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/chunk"
)

func Test_copy2trg(t *testing.T) {
	t.Run("copy ok", func(t *testing.T) {
		srcdir := t.TempDir()
		trgdir := t.TempDir()

		if err := os.WriteFile(filepath.Join(srcdir, "test.txt"), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
		trgfs := fsadapter.NewDirectory(trgdir)
		srcfs := os.DirFS(srcdir)

		if err := copy2trg(trgfs, "test-copy.txt", srcfs, "test.txt"); err != nil {
			t.Fatal(err)
		}
		// validate
		data, err := os.ReadFile(filepath.Join(trgdir, "test-copy.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "test" {
			t.Fatal("unexpected data")
		}
	})
	t.Run("copy fails", func(t *testing.T) {
		srcdir := t.TempDir()
		trgdir := t.TempDir()

		srcfs := os.DirFS(srcdir)
		trgfs := fsadapter.NewDirectory(trgdir)
		// source file does not exist.
		if err := copy2trg(trgfs, "test-copy.txt", srcfs, "test.txt"); err == nil {
			t.Fatal("expected error, but got nil")
		}
	})
}

func Test_avatarcopywrapper_copyAvatar(t *testing.T) {
	testUser := slack.User{
		ID:      "U12345678",
		Profile: slack.UserProfile{ImageOriginal: "https://example.com/avatar.jpg"},
	}
	type args struct {
		u slack.User
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(mfsa *mock_fsadapter.MockFSCloser, mavst *mock_source.MockStorage)
		wantErr  bool
	}{
		{
			name: "copies avatar",
			args: args{u: testUser},
			expectFn: func(mfsa *mock_fsadapter.MockFSCloser, mavst *mock_source.MockStorage) {
				tmpfile, err := os.CreateTemp(t.TempDir(), "")
				if err != nil {
					t.Fatal(err)
				}

				mockfs := fstest.MapFS{
					chunk.AvatarsDir + "/U12345678/avatar.jpg": {
						Data: []byte("avatar"),
					},
				}

				mavst.EXPECT().FS().Return(mockfs).Times(1)
				const avpath = chunk.AvatarsDir + "/U12345678/avatar.jpg"
				mavst.EXPECT().File(gomock.Any(), gomock.Any()).Return(avpath, nil).Times(1)
				mfsa.EXPECT().Create(avpath).Return(tmpfile, nil).Times(1)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mfsa := mock_fsadapter.NewMockFSCloser(ctrl)
			mavst := mock_source.NewMockStorage(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(mfsa, mavst)
			}
			a := &avatarcopywrapper{
				fsa:  mfsa,
				avst: mavst,
			}
			if err := a.copyAvatar(tt.args.u); (err != nil) != tt.wantErr {
				t.Errorf("avatarcopywrapper.copyAvatar() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
