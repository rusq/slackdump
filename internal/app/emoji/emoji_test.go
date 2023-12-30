package emoji

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/rusq/slackdump/v2/fsadapter"
	"go.uber.org/mock/gomock"
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

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Test_fetchEmoji(t *testing.T) {
	type args struct {
		ctx  context.Context
		dir  string
		name string
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
			args{context.Background(), "test", "file"},
			serverOptions{status: http.StatusOK, body: []byte("test data")},
			false,
			true,
			[]byte("test data"),
		},
		{
			"404",
			args{context.Background(), "test", "file"},
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

			if err := fetchEmoji(tt.args.ctx, fsa, tt.args.dir, tt.args.name, server.URL); (err != nil) != tt.wantErr {
				t.Errorf("fetch() error = %v, wantErr %v", err, tt.wantErr)
			}

			testfile := filepath.Join(dir, tt.args.dir, tt.args.name+".png")
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

func testEmojiC(emojis []emoji, wantClosed bool) <-chan emoji {
	ch := make(chan emoji)
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
		emojiC <-chan emoji
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
				ctx:    context.Background(),
				emojiC: testEmojiC([]emoji{{"test", "passed"}}, true),
			},
			func(ctx context.Context, fsa fsadapter.FS, dir string, name string, uri string) error {
				return nil
			},
			[]result{
				{name: "test"},
			},
		},
		{
			"cancelled context",
			args{
				ctx:    func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
				emojiC: testEmojiC([]emoji{}, false),
			},
			func(ctx context.Context, fsa fsadapter.FS, dir string, name string, uri string) error {
				return nil
			},
			[]result{
				{name: "", err: context.Canceled},
			},
		},
		{
			"fetch error",
			args{
				ctx:    context.Background(),
				emojiC: testEmojiC([]emoji{{"test", "passed"}}, true),
			},
			func(ctx context.Context, fsa fsadapter.FS, dir string, name string, uri string) error {
				return io.EOF
			},
			[]result{
				{name: "test", err: io.EOF},
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
			if !reflect.DeepEqual(results, tt.wantResult) {
				t.Errorf("results mismatch:\n\twant=%v\n\tgot =%v", tt.wantResult, results)
			}
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

	err := fetch(context.Background(), fsa, emojis, true)
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
	var chars = []rune("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

	b := make([]rune, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func Test_download(t *testing.T) {
	tmpdir := t.TempDir()

	type args struct {
		ctx      context.Context
		output   string
		failFast bool
	}
	tests := []struct {
		name    string
		args    args
		fetchFn fetchFunc
		expect  func(m *Mockemojidumper)
		wantErr bool
	}{
		{
			"save to directory",
			args{
				ctx:      context.Background(),
				output:   tmpdir,
				failFast: true,
			},
			emptyFetchFn,
			func(m *Mockemojidumper) {
				m.EXPECT().
					DumpEmojis(gomock.Any()).
					Return(map[string]string{
						"test": "https://blahblah.png",
					}, nil)
			},
			false,
		},
		{
			"save to zip file",
			args{
				ctx:      context.Background(),
				output:   filepath.Join(tmpdir, "test.zip"),
				failFast: true,
			},
			emptyFetchFn,
			func(m *Mockemojidumper) {
				m.EXPECT().
					DumpEmojis(gomock.Any()).
					Return(map[string]string{
						"test": "https://blahblah.png",
					}, nil)
			},
			false,
		},
		{
			"fails on fetch error with fail fast",
			args{
				ctx:      context.Background(),
				output:   tmpdir,
				failFast: true,
			},
			errorFetchFn,
			func(m *Mockemojidumper) {
				m.EXPECT().
					DumpEmojis(gomock.Any()).
					Return(map[string]string{
						"test": "https://blahblah.png",
					}, nil)
			},
			true,
		},
		{
			"fails on DumpEmojis error",
			args{
				ctx:      context.Background(),
				output:   tmpdir,
				failFast: false,
			},
			errorFetchFn,
			func(m *Mockemojidumper) {
				m.EXPECT().
					DumpEmojis(gomock.Any()).
					Return(nil, errors.New("no emojis for you, it's 1991."))
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setGlobalFetchFn(tt.fetchFn)
			sess := NewMockemojidumper(gomock.NewController(t))
			tt.expect(sess)
			if err := download(tt.args.ctx, sess, tt.args.output, tt.args.failFast); (err != nil) != tt.wantErr {
				t.Errorf("download() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
