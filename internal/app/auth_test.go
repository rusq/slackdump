package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rusq/slackdump/v2/auth"
	"github.com/rusq/slackdump/v2/internal/mocks/mock_app"
	"github.com/rusq/slackdump/v2/internal/mocks/mock_io"
	"github.com/stretchr/testify/assert"
)

func Test_isExistingFile(t *testing.T) {
	testfile := filepath.Join(t.TempDir(), "cookies.txt")
	if err := os.WriteFile(testfile, []byte("blah"), 0600); err != nil {
		t.Fatal(err)
	}

	type args struct {
		cookie string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"not a file", args{"$blah"}, false},
		{"file", args{testfile}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExistingFile(tt.args.cookie); got != tt.want {
				t.Errorf("isExistingFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackCreds_Type(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "fake_cookie")
	if err := os.WriteFile(testFile, []byte("unittest"), 0644); err != nil {
		t.Fatal(err)
	}
	type fields struct {
		Token  string
		Cookie string
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    auth.Type
		wantErr bool
	}{
		{"browser", fields{Token: "", Cookie: ""}, args{context.Background()}, auth.TypeBrowser, false},
		{"value", fields{Token: "t", Cookie: "c"}, args{context.Background()}, auth.TypeValue, false},
		{"browser", fields{Token: "t", Cookie: testFile}, args{context.Background()}, auth.TypeCookieFile, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := SlackCreds{
				Token:  tt.fields.Token,
				Cookie: tt.fields.Cookie,
			}
			got, err := c.Type(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("SlackCreds.Type() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SlackCreds.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlackCreds_IsEmpty(t *testing.T) {
	type fields struct {
		Token  string
		Cookie string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"empty", fields{Token: "", Cookie: ""}, true},
		{"empty", fields{Token: "x", Cookie: ""}, true},
		{"empty", fields{Token: "", Cookie: "x"}, true},
		{"empty", fields{Token: "x", Cookie: "x"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := SlackCreds{
				Token:  tt.fields.Token,
				Cookie: tt.fields.Cookie,
			}
			if got := c.IsEmpty(); got != tt.want {
				t.Errorf("SlackCreds.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func fakeAuthTester(retErr error) func(context.Context, auth.Provider) error {
	return func(ctx context.Context, p auth.Provider) error {
		return retErr
	}
}

func TestInitProvider(t *testing.T) {
	// prep
	testDir := t.TempDir()

	storedProv, _ := auth.NewValueAuth("xoxc", "xoxd")
	returnedProv, _ := auth.NewValueAuth("a", "b")
	// using default filer

	type args struct {
		ctx       context.Context
		cacheDir  string
		workspace string
	}
	tests := []struct {
		name        string
		args        args
		expect      func(m *mock_app.MockCredentials)
		authTestErr error
		want        auth.Provider
		wantErr     bool
	}{
		{
			"empty creds, no errors",
			args{context.Background(), testDir, "wsp"},
			func(m *mock_app.MockCredentials) {
				m.EXPECT().IsEmpty().Return(false)
				m.EXPECT().
					AuthProvider(gomock.Any(), "wsp").
					Return(storedProv, nil)
			},
			nil, //not used in the test
			storedProv,
			false,
		},
		{
			"creds empty, tryLoad succeeds",
			args{context.Background(), testDir, "wsp"},
			func(m *mock_app.MockCredentials) {
				m.EXPECT().IsEmpty().Return(true)
			},
			nil,
			storedProv, // loaded from file
			false,
		},
		{
			"creds empty, tryLoad fails",
			args{context.Background(), testDir, "wsp"},
			func(m *mock_app.MockCredentials) {
				m.EXPECT().IsEmpty().Return(true)
				m.EXPECT().AuthProvider(gomock.Any(), "wsp").Return(returnedProv, nil)
			},
			errors.New("auth test fail"), // auth test fails
			returnedProv,
			false,
		},
		{
			"creds non-empty, provider failed",
			args{context.Background(), testDir, "wsp"},
			func(m *mock_app.MockCredentials) {
				m.EXPECT().IsEmpty().Return(false)
				m.EXPECT().AuthProvider(gomock.Any(), "wsp").Return(nil, errors.New("authProvider failed"))
			},
			nil,
			nil,
			true,
		},
		{
			"creds non-empty, provider succeeds, save succeeds",
			args{context.Background(), testDir, "wsp"},
			func(m *mock_app.MockCredentials) {
				m.EXPECT().IsEmpty().Return(false)
				m.EXPECT().AuthProvider(gomock.Any(), "wsp").Return(returnedProv, nil)
			},
			nil,
			returnedProv,
			false,
		},
		{
			"creds non-empty, provider succeeds, save fails",
			args{context.Background(), t.TempDir() + "$", "wsp"},
			func(m *mock_app.MockCredentials) {
				m.EXPECT().IsEmpty().Return(false)
				m.EXPECT().AuthProvider(gomock.Any(), "wsp").Return(returnedProv, nil)
			},
			nil,
			returnedProv,
			false, // save error is ignored, and is visible only in trace.
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup
			oldTester := authTester
			defer func() {
				authTester = oldTester
			}()
			authTester = fakeAuthTester(tt.authTestErr)

			// resetting credentials
			credsFile := filepath.Join(testDir, credsFile)
			if err := saveCreds(filer, credsFile, storedProv); err != nil {
				t.Fatal(err)
			}

			mc := mock_app.NewMockCredentials(gomock.NewController(t))
			tt.expect(mc)

			// test
			got, err := InitProvider(tt.args.ctx, tt.args.cacheDir, tt.args.workspace, mc)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_tryLoad(t *testing.T) {
	// preparing file for testing
	testDir := t.TempDir()
	testProvider, _ := auth.NewValueAuth("xoxc", "xoxd")
	credsFile := filepath.Join(testDir, credsFile)
	if err := saveCreds(filer, credsFile, testProvider); err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx      context.Context
		filename string
	}
	tests := []struct {
		name        string
		args        args
		authTestErr error
		want        auth.Provider
		wantErr     bool
	}{
		{
			"all ok",
			args{context.Background(), credsFile},
			nil,
			testProvider,
			false,
		},
		{
			"load fails",
			args{context.Background(), filepath.Join(testDir, "fake")},
			nil,
			nil,
			true,
		},
		{
			"auth test fails",
			args{context.Background(), credsFile},
			errors.New("auth test fail"),
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup
			oldTester := authTester
			defer func() {
				authTester = oldTester
			}()
			authTester = fakeAuthTester(tt.authTestErr)

			got, err := tryLoad(tt.args.ctx, tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("tryLoad() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("tryLoad() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loadCreds(t *testing.T) {
	testProv, _ := auth.NewValueAuth("xoxc", "xoxd")
	var buf bytes.Buffer
	if err := auth.Save(&buf, testProv); err != nil {
		t.Fatal(err)
	}
	var testProvBytes = buf.Bytes()

	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		expect  func(mco *mock_app.MockcreateOpener, mrc *mock_io.MockReadCloser)
		want    auth.Provider
		wantErr bool
	}{
		{
			"all ok",
			args{"fakefile.ext"},
			func(mco *mock_app.MockcreateOpener, mrc *mock_io.MockReadCloser) {
				readCall := mrc.EXPECT().
					Read(gomock.Any()).
					DoAndReturn(func(b []byte) (int, error) {
						return copy(b, []byte(testProvBytes)), nil
					})
				mrc.EXPECT().Close().After(readCall).Return(nil)

				mco.EXPECT().
					Open("fakefile.ext").
					Return(mrc, nil)
			},
			testProv,
			false,
		},
		{
			"auth.Read error",
			args{"fakefile.ext"},
			func(mco *mock_app.MockcreateOpener, mrc *mock_io.MockReadCloser) {
				readCall := mrc.EXPECT().
					Read(gomock.Any()).
					Return(0, errors.New("auth.Read error"))
				mrc.EXPECT().Close().After(readCall).Return(nil)

				mco.EXPECT().
					Open("fakefile.ext").
					Return(mrc, nil)
			},
			auth.ValueAuth{},
			true,
		},
		{
			"read error",
			args{"fakefile.ext"},
			func(mco *mock_app.MockcreateOpener, mrc *mock_io.MockReadCloser) {
				mco.EXPECT().
					Open("fakefile.ext").
					Return(nil, errors.New("it was at this moment that test framework knew:  it fucked up"))
			},
			nil,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mco := mock_app.NewMockcreateOpener(ctrl)
			mrc := mock_io.NewMockReadCloser(ctrl)
			tt.expect(mco, mrc)

			got, err := loadCreds(mco, tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadCreds() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("loadCreds() = %v, want %v", got, tt.want)
			// }
		})
	}
}

func Test_saveCreds(t *testing.T) {
	testProv, _ := auth.NewValueAuth("xoxc", "xoxd")
	var buf bytes.Buffer
	if err := auth.Save(&buf, testProv); err != nil {
		t.Fatal(err)
	}
	var testProvBytes = buf.Bytes()

	type args struct {
		filename string
		p        auth.Provider
	}
	tests := []struct {
		name    string
		args    args
		expect  func(m *mock_app.MockcreateOpener, mwc *mock_io.MockWriteCloser)
		wantErr bool
	}{
		{
			"all ok",
			args{filename: "filename.ext", p: testProv},
			func(m *mock_app.MockcreateOpener, mwc *mock_io.MockWriteCloser) {
				wc := mwc.EXPECT().Write(testProvBytes).Return(len(testProvBytes), nil)
				mwc.EXPECT().Close().After(wc).Return(nil)

				m.EXPECT().Create("filename.ext").Return(mwc, nil)
			},
			false,
		},
		{
			"create fails",
			args{filename: "filename.ext", p: testProv},
			func(m *mock_app.MockcreateOpener, mwc *mock_io.MockWriteCloser) {
				m.EXPECT().Create("filename.ext").Return(nil, errors.New("create fail"))
			},
			true,
		},
		{
			"write fails",
			args{filename: "filename.ext", p: testProv},
			func(m *mock_app.MockcreateOpener, mwc *mock_io.MockWriteCloser) {
				wc := mwc.EXPECT().Write(testProvBytes).Return(0, errors.New("write fail"))
				mwc.EXPECT().Close().After(wc).Return(nil)

				m.EXPECT().Create("filename.ext").Return(mwc, nil)
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mco := mock_app.NewMockcreateOpener(ctrl)
			mwc := mock_io.NewMockWriteCloser(ctrl)
			tt.expect(mco, mwc)

			if err := saveCreds(mco, tt.args.filename, tt.args.p); (err != nil) != tt.wantErr {
				t.Errorf("saveCreds() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthReset(t *testing.T) {
	t.Run("file is removed", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, credsFile)
		if err := os.WriteFile(testFile, []byte("unit"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := AuthReset(tmpDir); err != nil {
			t.Errorf("AuthReset unexpected error: %s", err)
		}
		if fi, err := os.Stat(testFile); !os.IsNotExist(err) || fi != nil {
			t.Errorf("expected the %s to be removed, but it is there", testFile)
		}
	})
}

func Test_isWSL(t *testing.T) {
	tests := []struct {
		name         string
		wslDistroVal string
		want         bool
	}{
		{"yes WSL", "Ubuntu", true},
		{"not WSL", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("WSL_DISTRO_NAME", tt.wslDistroVal)
			defer os.Unsetenv("WSL_DISTRO_NAME")

			if got := isWSL(); got != tt.want {
				t.Errorf("isWSL() = %v, want %v", got, tt.want)
			}
		})
	}
}
