package dbase

import (
	"context"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/rusq/slack"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/chunk"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository"
	"github.com/rusq/slackdump/v3/internal/chunk/backend/dbase/repository/mock_repository"
	"github.com/rusq/slackdump/v3/internal/fixtures"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/internal/testutil"
)

var (
	testMsgChunk = &repository.DBChunk{
		ID:         1,
		UnixTS:     1234567890,
		TypeID:     chunk.CMessages,
		NumRecords: 2,
		ChannelID:  testutil.Ptr("C123456"),
		Final:      true,
	}

	testMsg1 = &slack.Message{
		Msg: slack.Msg{
			Timestamp: "1234567890.000000",
			Text:      "Hello",
			Files: []slack.File{
				{ID: "F123456", Name: "file1.txt", URLPrivate: "https://example.com/file1.txt"},
				{ID: "F123457", Name: "file2.txt", URLPrivate: "https://example.com/file2.txt"},
			},
		},
	}
	testMsg2 = &slack.Message{
		Msg: slack.Msg{
			Timestamp: "1234567891.000000",
			Text:      "World",
			Files: []slack.File{
				{ID: "F123458", Name: "file3.txt", URLPrivate: "https://example.com/file3.txt"},
				{ID: "F123459", Name: "file4.txt", URLPrivate: "https://example.com/file4.txt"},
			},
		},
	}
)

func Test_asmMessages(t *testing.T) {
	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockMessageRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:     context.Background(),
				conn:    nil, // not used
				dbchunk: testMsgChunk,
			},
			expectFn: func(m *mock_repository.MockMessageRepository) {
				it := testutil.Slice2Seq2([]repository.DBMessage{
					{
						ID:        1234567890,
						ChannelID: "C123456",
						TS:        "1234567890",
						Text:      "Hello",
						Data:      testutil.MarshalJSON(t, testMsg1),
					},
					{
						ID:        1234567891,
						ChannelID: "C123456",
						TS:        "1234567891",
						Text:      "World",
						Data:      testutil.MarshalJSON(t, testMsg2),
					},
				})
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), testMsgChunk.ID).Return(it, nil)
			},
			want: &chunk.Chunk{
				Type:      chunk.CMessages,
				ChannelID: "C123456",
				Timestamp: 1234567890,
				Count:     2,
				IsLast:    true,
				Messages: []slack.Message{
					*testMsg1,
					*testMsg2,
				},
			},
			wantErr: false,
		},
		{
			name: "error",
			args: args{
				ctx:     context.Background(),
				conn:    nil, // not used
				dbchunk: testMsgChunk,
			},
			expectFn: func(m *mock_repository.MockMessageRepository) {
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), testMsgChunk.ID).Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "iterator error",
			args: args{
				ctx:     context.Background(),
				conn:    nil, // not used
				dbchunk: testMsgChunk,
			},
			expectFn: func(m *mock_repository.MockMessageRepository) {
				it := testutil.ToIter([]testutil.TestResult[repository.DBMessage]{
					{
						Err: nil,
						V: repository.DBMessage{
							ID:        1234567890,
							ChannelID: "C123456",
							TS:        "1234567890",
							Text:      "Hello",
							Data:      testutil.MarshalJSON(t, testMsg1),
						},
					},
					{
						Err: assert.AnError,
						V:   repository.DBMessage{},
					},
				})
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), testMsgChunk.ID).Return(it, nil)
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpMsg
			t.Cleanup(func() {
				rpMsg = old
			})
			ctrl := gomock.NewController(t)
			rmm := mock_repository.NewMockMessageRepository(ctrl)
			rpMsg = rmm
			if tt.expectFn != nil {
				tt.expectFn(rmm)
			}

			got, err := asmMessages(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asmMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getMessage(t *testing.T) {
	type args struct {
		ctx  context.Context
		conn sqlx.ExtContext
		id   int64
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockMessageRepository)
		want     *slack.Message
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				id:   1234567890,
			},
			expectFn: func(m *mock_repository.MockMessageRepository) {
				m.EXPECT().Get(gomock.Any(), gomock.Any(), int64(1234567890)).Return(repository.DBMessage{
					ID:        1234567890,
					ChannelID: "C123456",
					TS:        "1234567890",
					Text:      "Hello",
					Data:      testutil.MarshalJSON(t, testMsg1),
				}, nil)
			},
			want:    testMsg1,
			wantErr: false,
		},
		{
			name: "error",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				id:   1234567890,
			},
			expectFn: func(m *mock_repository.MockMessageRepository) {
				m.EXPECT().Get(gomock.Any(), gomock.Any(), int64(1234567890)).Return(repository.DBMessage{}, assert.AnError)
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpMsg
			t.Cleanup(func() {
				rpMsg = old
			})
			ctrl := gomock.NewController(t)
			rmm := mock_repository.NewMockMessageRepository(ctrl)
			rpMsg = rmm
			if tt.expectFn != nil {
				tt.expectFn(rmm)
			}
			got, err := getMessage(tt.args.ctx, tt.args.conn, tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("getMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_asmThreadMessages(t *testing.T) {
	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockMessageRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:     context.Background(),
				conn:    nil, // not used
				dbchunk: testMsgChunk,
			},
			expectFn: func(m *mock_repository.MockMessageRepository) {
				it := testutil.Slice2Seq2([]repository.DBMessage{
					{
						ID:        1234567890,
						ChannelID: "C123456",
						TS:        "1234567890",
						Text:      "Hello",
						Data:      testutil.MarshalJSON(t, testMsg1),
					},
					{
						ID:        1234567891,
						ChannelID: "C123456",
						TS:        "1234567891",
						Text:      "World",
						Data:      testutil.MarshalJSON(t, testMsg2),
					},
				})
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), testMsgChunk.ID).Return(it, nil)
			},
			want: &chunk.Chunk{
				Type:      chunk.CMessages,
				ChannelID: "C123456",
				Timestamp: 1234567890,
				Count:     2,
				IsLast:    true,
				Messages: []slack.Message{
					*testMsg1,
					*testMsg2,
				},
			},
			wantErr: false,
		},
		{
			name: "error",
			args: args{
				ctx:     context.Background(),
				conn:    nil, // not used
				dbchunk: testMsgChunk,
			},
			expectFn: func(m *mock_repository.MockMessageRepository) {
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), testMsgChunk.ID).Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "iterator error",
			args: args{
				ctx:     context.Background(),
				conn:    nil, // not used
				dbchunk: testMsgChunk,
			},
			expectFn: func(m *mock_repository.MockMessageRepository) {
				it := testutil.ToIter([]testutil.TestResult[repository.DBMessage]{
					{
						Err: nil,
						V: repository.DBMessage{
							ID:        1234567890,
							ChannelID: "C123456",
							TS:        "1234567890",
							Text:      "Hello",
							Data:      testutil.MarshalJSON(t, testMsg1),
						},
					},
					{
						Err: assert.AnError,
						V:   repository.DBMessage{},
					},
				})
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), testMsgChunk.ID).Return(it, nil)
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpMsg
			t.Cleanup(func() {
				rpMsg = old
			})
			ctrl := gomock.NewController(t)
			rmm := mock_repository.NewMockMessageRepository(ctrl)
			rpMsg = rmm
			if tt.expectFn != nil {
				tt.expectFn(rmm)
			}
			got, err := asmThreadMessages(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmThreadMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asmThreadMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_asmFiles(t *testing.T) {
	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockFileRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				dbchunk: &repository.DBChunk{
					ID:         1,
					TypeID:     chunk.CFiles,
					NumRecords: 1,
					ChannelID:  testutil.Ptr("C123456"),
					UnixTS:     1234567890,
				},
			},
			expectFn: func(m *mock_repository.MockFileRepository) {
				it := testutil.Slice2Seq2([]repository.DBFile{
					{
						ID:        "F123456",
						ChunkID:   1,
						ChannelID: "C123456",
						Data:      testutil.MarshalJSON(t, testMsg1.Files[0]),
						Index:     0,
					},
				})
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), testMsgChunk.ID).Return(it, nil)
			},
			want: &chunk.Chunk{
				Type:      chunk.CFiles,
				ChannelID: "C123456",
				Timestamp: 1234567890,
				Count:     1,
				Files: []slack.File{
					testMsg1.Files[0],
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpFile
			t.Cleanup(func() {
				rpFile = old
			})
			ctrl := gomock.NewController(t)
			rmm := mock_repository.NewMockFileRepository(ctrl)
			rpFile = rmm
			if tt.expectFn != nil {
				tt.expectFn(rmm)
			}
			got, err := asmFiles(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asmFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_asmUsers(t *testing.T) {
	testUsers := fixtures.Load[[]slack.User](string(fixtures.TestExpUsersJSON))
	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockUserRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				dbchunk: &repository.DBChunk{
					ID:         1,
					TypeID:     chunk.CUsers,
					UnixTS:     1234567890,
					NumRecords: int32(len(testUsers)),
				},
			},
			expectFn: func(m *mock_repository.MockUserRepository) {
				dbu := make([]repository.DBUser, len(testUsers))
				for i, u := range testUsers {
					du, err := repository.NewDBUser(1, i, &u)
					if err != nil {
						t.Fatalf("failed to create DBUser: %v", err)
					}
					dbu[i] = *du
				}
				it := testutil.Slice2Seq2(dbu)
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), int64(1)).Return(it, nil)
			},
			want: &chunk.Chunk{
				Type:      chunk.CUsers,
				Timestamp: 1234567890,
				Count:     int32(len(testUsers)),
				Users:     testUsers,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpUser
			t.Cleanup(func() {
				rpUser = old
			})
			ctrl := gomock.NewController(t)
			rmu := mock_repository.NewMockUserRepository(ctrl)
			rpUser = rmu
			if tt.expectFn != nil {
				tt.expectFn(rmu)
			}
			got, err := asmUsers(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asmUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_asmChannels(t *testing.T) {
	channels := fixtures.Load[[]slack.Channel](string(fixtures.TestExpChannelsJSON))
	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockChannelRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				dbchunk: &repository.DBChunk{
					ID:         1,
					TypeID:     chunk.CChannels,
					UnixTS:     1234567890,
					NumRecords: int32(len(channels)),
				},
			},
			expectFn: func(m *mock_repository.MockChannelRepository) {
				dbc := make([]repository.DBChannel, len(channels))
				for i, c := range channels {
					dc, err := repository.NewDBChannel(1, i, &c)
					if err != nil {
						t.Fatalf("failed to create DBChannel: %v", err)
					}
					dbc[i] = *dc
				}
				it := testutil.Slice2Seq2(dbc)
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), int64(1)).Return(it, nil)
			},
			want: &chunk.Chunk{
				Type:      chunk.CChannels,
				Timestamp: 1234567890,
				Count:     int32(len(channels)),
				Channels:  channels,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpChan
			t.Cleanup(func() {
				rpChan = old
			})
			ctrl := gomock.NewController(t)
			rmc := mock_repository.NewMockChannelRepository(ctrl)
			rpChan = rmc
			if tt.expectFn != nil {
				tt.expectFn(rmc)
			}
			got, err := asmChannels(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmChannels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asmChannels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_asmChannelInfo(t *testing.T) {
	channelinfo := structures.ChannelFromID("C123456")
	channelinfo.Name = "test-channel"
	channelinfo.IsChannel = true
	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockChannelRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				dbchunk: &repository.DBChunk{
					ID:        1,
					TypeID:    chunk.CChannelInfo,
					UnixTS:    1234567890,
					ChannelID: testutil.Ptr("C123456"),
				},
			},
			expectFn: func(m *mock_repository.MockChannelRepository) {
				m.EXPECT().OneForChunk(gomock.Any(), gomock.Any(), int64(1)).Return(repository.DBChannel{
					ID:   "C123456",
					Data: testutil.MarshalJSON(t, channelinfo),
				}, nil)
			},
			want: &chunk.Chunk{
				Type:      chunk.CChannelInfo,
				Timestamp: 1234567890,
				ChannelID: "C123456",
				Channel:   channelinfo,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpChan
			t.Cleanup(func() {
				rpChan = old
			})
			ctrl := gomock.NewController(t)
			rmc := mock_repository.NewMockChannelRepository(ctrl)
			rpChan = rmc
			if tt.expectFn != nil {
				tt.expectFn(rmc)
			}
			got, err := asmChannelInfo(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmChannelInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_asmWorkspaceInfo(t *testing.T) {
	workspaceInfo := &slack.AuthTestResponse{
		URL:          "https://example.com",
		Team:         "Test Team",
		User:         "Test User",
		TeamID:       "T123456",
		UserID:       "U123456",
		EnterpriseID: "E123456",
		BotID:        "B123456",
	}

	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockWorkspaceRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				dbchunk: &repository.DBChunk{
					ID:     1,
					TypeID: chunk.CWorkspaceInfo,
					UnixTS: 1234567890,
				},
			},
			expectFn: func(m *mock_repository.MockWorkspaceRepository) {
				m.EXPECT().OneForChunk(gomock.Any(), gomock.Any(), int64(1)).Return(repository.DBWorkspace{
					ID:   1,
					Data: testutil.MarshalJSON(t, workspaceInfo),
				}, nil)
			},
			want: &chunk.Chunk{
				Type:          chunk.CWorkspaceInfo,
				Timestamp:     1234567890,
				WorkspaceInfo: workspaceInfo,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpWsp
			t.Cleanup(func() {
				rpWsp = old
			})
			ctrl := gomock.NewController(t)
			rmw := mock_repository.NewMockWorkspaceRepository(ctrl)
			rpWsp = rmw
			if tt.expectFn != nil {
				tt.expectFn(rmw)
			}
			got, err := asmWorkspaceInfo(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmWorkspaceInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asmWorkspaceInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_asmChannelUsers(t *testing.T) {
	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockChannelUserRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				dbchunk: &repository.DBChunk{
					ID:         1,
					TypeID:     chunk.CChannelUsers,
					UnixTS:     1234567890,
					NumRecords: 3,
				},
			},
			expectFn: func(m *mock_repository.MockChannelUserRepository) {
				it := testutil.Slice2Seq2([]repository.DBChannelUser{
					{
						ChannelID: "C123456",
						UserID:    "U111",
						ChunkID:   1,
						Index:     0,
					},
					{
						ChannelID: "C123456",
						UserID:    "U222",
						ChunkID:   1,
						Index:     1,
					},
					{
						ChannelID: "C123456",
						UserID:    "U333",
						ChunkID:   1,
						Index:     2,
					},
				})
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), int64(1)).Return(it, nil)
			},
			want: &chunk.Chunk{
				Type:         chunk.CChannelUsers,
				Timestamp:    1234567890,
				Count:        3,
				ChannelUsers: []string{"U111", "U222", "U333"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpChanUser
			t.Cleanup(func() {
				rpChanUser = old
			})
			ctrl := gomock.NewController(t)
			rmcu := mock_repository.NewMockChannelUserRepository(ctrl)
			rpChanUser = rmcu
			if tt.expectFn != nil {
				tt.expectFn(rmcu)
			}
			got, err := asmChannelUsers(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmChannelUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asmChannelUsers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_asmSearchMessages(t *testing.T) {
	sms := []slack.SearchMessage{
		{
			Channel:   slack.CtxChannel{},
			User:      "U1111",
			Username:  "Test User",
			Timestamp: "1234567890.000000",
			Blocks:    slack.Blocks{},
			Text:      "Hello",
		},
		{
			Channel:   slack.CtxChannel{},
			User:      "U2222",
			Username:  "Test User 2",
			Timestamp: "1234567890.000000",
			Blocks:    slack.Blocks{},
			Text:      "Hello",
		},
	}
	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockSearchMessageRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				dbchunk: &repository.DBChunk{
					ID:          1,
					TypeID:      chunk.CSearchMessages,
					UnixTS:      1234567890,
					NumRecords:  int32(len(sms)),
					SearchQuery: testutil.Ptr("test"),
				},
			},
			expectFn: func(m *mock_repository.MockSearchMessageRepository) {
				it := testutil.Slice2Seq2([]repository.DBSearchMessage{
					{
						ID:        1234567890,
						ChannelID: "C123456",
						TS:        "1234567890",
						Text:      testutil.Ptr("Hello"),
						Data:      testutil.MarshalJSON(t, sms[0]),
					},
					{
						ID:        1234567891,
						ChannelID: "C123456",
						TS:        "1234567891",
						Text:      testutil.Ptr("World"),
						Data:      testutil.MarshalJSON(t, sms[1]),
					},
				})
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), int64(1)).Return(it, nil)
			},
			want: &chunk.Chunk{
				Type:        chunk.CSearchMessages,
				Timestamp:   1234567890,
				Count:       int32(len(sms)),
				SearchQuery: "test",
				SearchMessages: []slack.SearchMessage{
					sms[0],
					sms[1],
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpSrchMsg
			t.Cleanup(func() {
				rpSrchMsg = old
			})
			ctrl := gomock.NewController(t)
			rmsm := mock_repository.NewMockSearchMessageRepository(ctrl)
			rpSrchMsg = rmsm
			if tt.expectFn != nil {
				tt.expectFn(rmsm)
			}
			got, err := asmSearchMessages(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmSearchMessages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asmSearchMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_asmSearchFiles(t *testing.T) {
	sfs := []slack.File{
		{ID: "F123456", Name: "file1.txt", URLPrivate: "https://example.com/file1.txt"},
		{ID: "F123457", Name: "file2.txt", URLPrivate: "https://example.com/file2.txt"},
	}

	type args struct {
		ctx     context.Context
		conn    sqlx.ExtContext
		dbchunk *repository.DBChunk
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(m *mock_repository.MockSearchFileRepository)
		want     *chunk.Chunk
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:  context.Background(),
				conn: nil, // not used
				dbchunk: &repository.DBChunk{
					ID:          1,
					TypeID:      chunk.CSearchFiles,
					UnixTS:      1234567890,
					NumRecords:  int32(len(sfs)),
					SearchQuery: testutil.Ptr("test"),
				},
			},
			expectFn: func(m *mock_repository.MockSearchFileRepository) {
				it := testutil.Slice2Seq2([]repository.DBSearchFile{
					{
						ID:      1,
						ChunkID: 1,
						FileID:  "F123456",
						Index:   0,
						Data:    testutil.MarshalJSON(t, sfs[0]),
					},
					{
						ID:      2,
						ChunkID: 1,
						FileID:  "F123457",
						Index:   1,
						Data:    testutil.MarshalJSON(t, sfs[1]),
					},
				})
				m.EXPECT().AllForChunk(gomock.Any(), gomock.Any(), int64(1)).Return(it, nil)
			},
			want: &chunk.Chunk{
				Type:        chunk.CSearchFiles,
				Timestamp:   1234567890,
				Count:       int32(len(sfs)),
				SearchQuery: "test",
				SearchFiles: []slack.File{
					sfs[0],
					sfs[1],
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := rpSrchFile
			t.Cleanup(func() {
				rpSrchFile = old
			})
			ctrl := gomock.NewController(t)
			rmsf := mock_repository.NewMockSearchFileRepository(ctrl)
			rpSrchFile = rmsf
			if tt.expectFn != nil {
				tt.expectFn(rmsf)
			}
			got, err := asmSearchFiles(tt.args.ctx, tt.args.conn, tt.args.dbchunk)
			if (err != nil) != tt.wantErr {
				t.Errorf("asmSearchFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("asmSearchFiles() = %v, want %v", got, tt.want)
			}
		})
	}
}
