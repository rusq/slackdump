package emojidl

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/rusq/fsadapter"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/edge"
)

type fetchFunc func(ctx context.Context, fsa fsadapter.FS, dir string, name string, uri string) error

var mu sync.Mutex // globals mutex

var (
	emptyFetchFn = func(ctx context.Context, fsa fsadapter.FS, dir, name, uri string) error { return nil }
	errorFetchFn = func(ctx context.Context, fsa fsadapter.FS, dir, name, uri string) error {
		return errors.New("your shattered hopes")
	}
)

func setGlobalFetchFn(fn fetchFunc) {
	mu.Lock()
	defer mu.Unlock()
	fetchFn = fn
}

func Test_fetchEmoji(t *testing.T) {
	type args struct {
		ctx       context.Context
		dir       string
		name      string
		urlsuffix string
	}
	type serverOptions struct {
		status int
		body   []byte
	}
	tests := []struct {
		name          string
		args          args
		opts          serverOptions
		wantErr       bool
		wantFileExist bool
		wantFileData  []byte
	}{
		{
			"ok",
			args{t.Context(), "test", "file", "/somepath/file.png"},
			serverOptions{status: http.StatusOK, body: []byte("test data")},
			false,
			true,
			[]byte("test data"),
		},
		{
			"gif",
			args{t.Context(), "test", "file", "/somepath/file.gif"},
			serverOptions{status: http.StatusOK, body: []byte("test data")},
			false,
			true,
			[]byte("test data"),
		},
		{
			"404",
			args{t.Context(), "test", "file", "/somepath/file.png"},
			serverOptions{status: http.StatusNotFound, body: nil},
			true,
			true,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.opts.status)
				if _, err := w.Write(tt.opts.body); err != nil {
					panic(err)
				}
			}))
			defer server.Close()

			dir := t.TempDir()
			fsa, err := fsadapter.New(dir)
			if err != nil {
				t.Fatalf("failed to create test dir: %s", err)
			}

			if err := fetchEmoji(tt.args.ctx, fsa, tt.args.dir, tt.args.name, server.URL+tt.args.urlsuffix); (err != nil) != tt.wantErr {
				t.Errorf("fetch() error = %v, wantErr %v", err, tt.wantErr)
			}

			ext := path.Ext(tt.args.urlsuffix)
			testfile := filepath.Join(dir, tt.args.dir, tt.args.name+ext)
			_, err = os.Stat(testfile)
			if notExist := os.IsNotExist(err); notExist != !tt.wantFileExist {
				t.Errorf("os.IsNotExist=%v tt.wantFileExist=%v", notExist, tt.wantFileExist)
			}
			if !tt.wantFileExist {
				return
			}
			got, err := os.ReadFile(testfile)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, tt.wantFileData) {
				t.Errorf("file contents error: want=%v got=%v", tt.wantFileData, got)
			}
		})
	}
}

func testEmojiC(emojis []edge.Emoji, wantClosed bool) <-chan edge.Emoji {
	ch := make(chan edge.Emoji)
	go func() {
		for _, e := range emojis {
			ch <- e
		}
		if wantClosed {
			close(ch)
		}
	}()
	return ch
}

func Test_worker(t *testing.T) {
	type args struct {
		ctx    context.Context
		emojiC <-chan edge.Emoji
	}
	tests := []struct {
		name       string
		args       args
		fetchFn    fetchFunc
		wantResult []result
	}{
		{
			"all ok",
			args{
				ctx:    t.Context(),
				emojiC: testEmojiC([]edge.Emoji{{Name: "test", URL: "passed"}}, true),
			},
			func(ctx context.Context, fsa fsadapter.FS, dir string, name string, uri string) error {
				return nil
			},
			[]result{
				{emoji: edge.Emoji{Name: "test", URL: "passed"}},
			},
		},
		{
			"cancelled context",
			args{
				ctx:    func() context.Context { ctx, cancel := context.WithCancel(t.Context()); cancel(); return ctx }(),
				emojiC: testEmojiC([]edge.Emoji{}, false),
			},
			func(ctx context.Context, fsa fsadapter.FS, dir string, name string, uri string) error {
				return nil
			},
			[]result{
				{emoji: edge.Emoji{Name: ""}, err: context.Canceled},
			},
		},
		{
			"fetch error",
			args{
				ctx:    t.Context(),
				emojiC: testEmojiC([]edge.Emoji{{Name: "test", URL: "passed"}}, true),
			},
			func(ctx context.Context, fsa fsadapter.FS, dir string, name string, uri string) error {
				return io.EOF
			},
			[]result{
				{emoji: edge.Emoji{Name: "test", URL: "passed"}, err: io.EOF},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setGlobalFetchFn(tt.fetchFn)

			fsa, _ := fsadapter.New(t.TempDir())
			resultC := make(chan result)

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				worker(tt.args.ctx, fsa, tt.args.emojiC, resultC)
				wg.Done()
			}()
			go func() {
				wg.Wait()
				close(resultC)
			}()
			var results []result
			for r := range resultC {
				results = append(results, r)
			}
			assert.Equal(t, tt.wantResult, results)
		})
	}
}

func Test_fetch(t *testing.T) {
	emojis := generateEmojis(50)
	fsa, _ := fsadapter.New(t.TempDir())

	got := make(map[string]string, len(emojis))
	var mu sync.Mutex

	setGlobalFetchFn(func(ctx context.Context, fsa fsadapter.FS, dir string, name string, uri string) error {
		mu.Lock()
		got[name] = uri
		mu.Unlock()
		return nil
	})

	err := fetch(t.Context(), fsa, emojis, true, nil)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(emojis, got) {
		t.Error("emojis!=got")
	}
}

func generateEmojis(n int) (ret map[string]string) {
	ret = make(map[string]string, n)
	for i := 0; i < n; i++ {
		ret[randString(10)] = "https://emoji.slack.com/" + randString(20)
	}
	return
}

func randString(n int) string {
	chars := []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func Test_download(t *testing.T) {
	tmpdir := t.TempDir()

	type args struct {
		ctx    context.Context
		output string
		opts   *Options
	}
	tests := []struct {
		name    string
		args    args
		fetchFn fetchFunc
		expect  func(m *MockEmojiDumper)
		wantErr bool
	}{
		{
			"save to directory",
			args{
				ctx:    t.Context(),
				output: tmpdir,
				opts: &Options{
					FailFast:  true,
					WithFiles: true,
				},
			},
			emptyFetchFn,
			func(m *MockEmojiDumper) {
				m.EXPECT().
					GetEmojiContext(gomock.Any()).
					Return(map[string]string{
						"test": "https://blahblah.png",
					}, nil)
			},
			false,
		},
		{
			"save to zip file",
			args{
				ctx:    t.Context(),
				output: filepath.Join(tmpdir, "test.zip"),
				opts: &Options{
					FailFast:  true,
					WithFiles: true,
				},
			},
			emptyFetchFn,
			func(m *MockEmojiDumper) {
				m.EXPECT().
					GetEmojiContext(gomock.Any()).
					Return(map[string]string{
						"test": "https://blahblah.png",
					}, nil)
			},
			false,
		},
		{
			"fails on fetch error with fail fast",
			args{
				ctx:    t.Context(),
				output: tmpdir,
				opts: &Options{
					FailFast:  true,
					WithFiles: true,
				},
			},
			errorFetchFn,
			func(m *MockEmojiDumper) {
				m.EXPECT().
					GetEmojiContext(gomock.Any()).
					Return(map[string]string{
						"test": "https://blahblah.png",
					}, nil)
			},
			true,
		},
		{
			"fails on DumpEmojis error",
			args{
				ctx:    t.Context(),
				output: tmpdir,
				opts: &Options{
					FailFast:  false,
					WithFiles: true,
				},
			},
			errorFetchFn,
			func(m *MockEmojiDumper) {
				m.EXPECT().
					GetEmojiContext(gomock.Any()).
					Return(nil, errors.New("no emojis for you, it's 1991."))
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setGlobalFetchFn(tt.fetchFn)
			sess := NewMockEmojiDumper(gomock.NewController(t))
			tt.expect(sess)
			fs, err := fsadapter.New(tt.args.output)
			if err != nil {
				t.Fatal(err)
			}
			defer fs.Close()
			if err := DlFS(tt.args.ctx, sess, fs, tt.args.opts, nil); (err != nil) != tt.wantErr {
				t.Errorf("download() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
