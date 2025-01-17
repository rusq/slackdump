package cache

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/auth"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/mocks/mock_appauth"
	"github.com/rusq/slackdump/v3/internal/mocks/mock_io"
)

func Test_isExistingFile(t *testing.T) {
	testfile := filepath.Join(t.TempDir(), "cookies.txt")
	if err := os.WriteFile(testfile, []byte("blah"), 0o600); err != nil {
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

func TestAuthData_Type(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "fake_cookie")
	if err := os.WriteFile(testFile, []byte("unittest"), 0o644); err != nil {
		t.Fatal(err)
	}
	type fields struct {
		Token         string
		Cookie        string
		UsePlaywright bool
	}
	type args struct {
		ctx context.Context
	}
	type test struct {
		name    string
		fields  fields
		args    args
		want    AuthType
		wantErr bool
	}
	tests := []test{
		{"value", fields{Token: "t", Cookie: "c"}, args{context.Background()}, ATValue, false},
		{"cookie file", fields{Token: "t", Cookie: testFile}, args{context.Background()}, ATCookieFile, false},
	}
	if !isWSL {
		tests = append(tests, test{"rod", fields{Token: "", Cookie: ""}, args{context.Background()}, ATRod, false})
		tests = append(tests, test{"playwright", fields{Token: "", Cookie: "", UsePlaywright: true}, args{context.Background()}, ATPlaywright, false})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := AuthData{
				Token:         tt.fields.Token,
				Cookie:        tt.fields.Cookie,
				UsePlaywright: tt.fields.UsePlaywright,
			}
			got, err := c.Type(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthData.Type() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AuthData.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAuthData_IsEmpty(t *testing.T) {
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
		{"no token", fields{Token: "", Cookie: "x"}, true},
		{"xoxc: token and cookie present", fields{Token: fixtures.TestClientToken, Cookie: "x"}, false},
		{"xoxc: no cookie is not ok", fields{Token: fixtures.TestClientToken, Cookie: ""}, true},
		{"personal token: no cookie is ok", fields{Token: fixtures.TestPersonalToken, Cookie: ""}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := AuthData{
				Token:  tt.fields.Token,
				Cookie: tt.fields.Cookie,
			}
			if got := c.IsEmpty(); got != tt.want {
				t.Errorf("AuthData.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitProvider(t *testing.T) {
	// prep
	testDir := t.TempDir()

	storedProv, _ := auth.NewValueAuth("xoxc", "xoxd")
	returnedProv, _ := auth.NewValueAuth("a", "b")

	type args struct {
		ctx       context.Context
		cacheDir  string
		workspace string
	}
	tests := []struct {
		name        string
		args        args
		expect      func(m *mock_appauth.MockCredentials)
		authTestErr error
		want        auth.Provider
		wantErr     bool
	}{
		{
			"empty creds, no errors",
			args{context.Background(), testDir, "wsp"},
			func(m *mock_appauth.MockCredentials) {
				m.EXPECT().IsEmpty().Return(false)
				m.EXPECT().
					AuthProvider(gomock.Any(), "wsp").
					Return(storedProv, nil)
			},
			nil, // not used in the test
			storedProv,
			false,
		},
		{
			"creds empty, tryLoad succeeds",
			args{context.Background(), testDir, "wsp"},
			func(m *mock_appauth.MockCredentials) {
				m.EXPECT().IsEmpty().Return(true)
			},
			nil,
			storedProv, // loaded from file
			false,
		},
		{
			"creds empty, tryLoad fails",
			args{context.Background(), testDir, "wsp"},
			func(m *mock_appauth.MockCredentials) {
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
			func(m *mock_appauth.MockCredentials) {
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
			func(m *mock_appauth.MockCredentials) {
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
			func(m *mock_appauth.MockCredentials) {
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
			credsFile := filepath.Join(testDir, defCredsFile)
			container := encryptedFile{}
			if err := saveCreds(container, credsFile, storedProv); err != nil {
				t.Fatal(err)
			}

			mc := mock_appauth.NewMockCredentials(gomock.NewController(t))
			tt.expect(mc)

			auther := newAuthenticator(tt.args.cacheDir, "")
			// test
			got, err := auther.initProvider(tt.args.ctx, defCredsFile, tt.args.workspace, mc)
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

func fakeAuthTester(err error) func(_ auth.Provider, ctx context.Context) (*slack.AuthTestResponse, error) {
	return func(_ auth.Provider, ctx context.Context) (*slack.AuthTestResponse, error) {
		return nil, err
	}
}

func Test_tryLoad(t *testing.T) {
	// preparing file for testing
	testDir := t.TempDir()
	testProvider, _ := auth.NewValueAuth("xoxc", "xoxd")
	credsFile := filepath.Join(testDir, defCredsFile)

	filer := encryptedFile{}
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

			a := authenticator{
				ct: encryptedFile{},
			}

			got, err := a.tryLoad(tt.args.ctx, tt.args.filename)
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
	testProvBytes := buf.Bytes()

	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		expect  func(mco *mock_appauth.Mockcontainer, mrc *mock_io.MockReadCloser)
		want    auth.Provider
		wantErr bool
	}{
		{
			"all ok",
			args{"fakefile.ext"},
			func(mco *mock_appauth.Mockcontainer, mrc *mock_io.MockReadCloser) {
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
			func(mco *mock_appauth.Mockcontainer, mrc *mock_io.MockReadCloser) {
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
			func(mco *mock_appauth.Mockcontainer, mrc *mock_io.MockReadCloser) {
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
			mco := mock_appauth.NewMockcontainer(ctrl)
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
	testProvBytes := buf.Bytes()

	type args struct {
		filename string
		p        auth.Provider
	}
	tests := []struct {
		name    string
		args    args
		expect  func(m *mock_appauth.Mockcontainer, mwc *mock_io.MockWriteCloser)
		wantErr bool
	}{
		{
			"all ok",
			args{filename: "filename.ext", p: testProv},
			func(m *mock_appauth.Mockcontainer, mwc *mock_io.MockWriteCloser) {
				wc := mwc.EXPECT().Write(testProvBytes).Return(len(testProvBytes), nil)
				mwc.EXPECT().Close().After(wc).Return(nil)

				m.EXPECT().Create("filename.ext").Return(mwc, nil)
			},
			false,
		},
		{
			"create fails",
			args{filename: "filename.ext", p: testProv},
			func(m *mock_appauth.Mockcontainer, mwc *mock_io.MockWriteCloser) {
				m.EXPECT().Create("filename.ext").Return(nil, errors.New("create fail"))
			},
			true,
		},
		{
			"write fails",
			args{filename: "filename.ext", p: testProv},
			func(m *mock_appauth.Mockcontainer, mwc *mock_io.MockWriteCloser) {
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
			mco := mock_appauth.NewMockcontainer(ctrl)
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
		testFile := filepath.Join(tmpDir, defCredsFile)
		if err := os.WriteFile(testFile, []byte("unit"), 0o644); err != nil {
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

func Test_encryptedFile_Open(t *testing.T) {
	tmpdir := t.TempDir()
	mkfile := func(machineID string, contents []byte) string {
		c := encryptedFile{machineID: machineID}
		tf, err := os.CreateTemp(tmpdir, "")
		if err != nil {
			panic(err)
		}
		tf.Close()

		ef, err := c.Create(tf.Name())
		if err != nil {
			panic(err)
		}
		defer ef.Close()
		if _, err := io.Copy(ef, bytes.NewReader(contents)); err != nil {
			panic(err)
		}
		return tf.Name()
	}

	type fields struct {
		machineID string
	}
	type args struct {
		filename string
	}
	tests := []struct {
		name      string
		fields    fields
		contents  []byte
		args      args
		wantMatch bool
		wantErr   bool
	}{
		{
			"encrypted with the same machine ID",
			fields{
				machineID: "123",
			},
			[]byte("unit test"),
			args{mkfile("123", []byte("unit test"))},
			true,
			false,
		},
		{
			"different machine ID",
			fields{
				machineID: "123",
			},
			[]byte("unit test"),
			args{mkfile("456", []byte("unit test"))},
			false,
			false,
		},
		{
			"override vs real ID",
			fields{},
			[]byte("unit test"),
			args{mkfile("456", []byte("unit test"))},
			false,
			false,
		},
		{
			"machine ID",
			fields{},
			[]byte("unit test"),
			args{mkfile("", []byte("unit test"))},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := encryptedFile{
				machineID: tt.fields.machineID,
			}
			got, err := f.Open(tt.args.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("encryptedFile.Open() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			defer got.Close()
			var contents bytes.Buffer
			if _, err := io.Copy(&contents, got); err != nil {
				t.Fatalf("failed to read the test file: %s", err)
			}

			if bytes.Equal(contents.Bytes(), tt.contents) != tt.wantMatch {
				t.Errorf("encryptedFile.Open() = %#v, want %#v", contents.Bytes(), tt.contents)
			}
		})
	}
}
