package convert

import (
	"context"
	"iter"
	"slices"
	"testing"
	"time"

	"github.com/rusq/slackdump/v3/source/mock_source"

	"github.com/rusq/slack"
	"go.uber.org/mock/gomock"

	"github.com/rusq/slackdump/v3/internal/fasttime"
	"github.com/rusq/slackdump/v3/internal/structures"
	"github.com/rusq/slackdump/v3/mocks/mock_processor"
)

func Test_encodeMessages(t *testing.T) {
	type args struct {
		ctx context.Context
		// rec processor.Conversations
		// src source.Sourcer
		ch *slack.Channel
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(t *testing.T, mc *mock_processor.MockConversations, ms *mock_source.MockSourcer)
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx: t.Context(),
				ch:  structures.ChannelFromID("C123"),
			},
			expectFn: func(t *testing.T, mc *mock_processor.MockConversations, ms *mock_source.MockSourcer) {
				it, msgs := msgGenerator(t, time.Now().UnixMicro(), 203, defaultChunkSize)
				ms.EXPECT().AllMessages(gomock.Any(), "C123").Return(it, nil)
				for i := range len(msgs) - 1 { // this should be called 203/100=2 times
					mc.EXPECT().Messages(gomock.Any(), "C123", 0, false, msgs[i]).Return(nil)
				}
				// last flush call, called with the 3 message chunk
				mc.EXPECT().Messages(gomock.Any(), "C123", 0, true, msgs[len(msgs)-1]).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := mock_processor.NewMockConversations(ctrl)
			ms := mock_source.NewMockSourcer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(t, mc, ms)
			}
			if err := encodeMessages(tt.args.ctx, mc, ms, tt.args.ch); (err != nil) != tt.wantErr {
				t.Errorf("encodeMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func msgGenerator(t *testing.T, startTS int64, num int, chunkSz int) (iter.Seq2[slack.Message, error], [][]slack.Message) {
	// generating messages
	msg := make([]slack.Message, num)
	for i := range num {
		ts := fasttime.Int2TS(startTS + int64(i))
		m := slack.Message{
			Msg: slack.Msg{
				Timestamp: ts,
				Text:      "msg: " + ts,
			},
		}
		msg[i] = m
	}
	msgs := slices.Collect(slices.Chunk(msg, chunkSz))
	// creating iterator
	it := func(yield func(slack.Message, error) bool) {
		for _, m := range msg {
			if !yield(m, nil) {
				break
			}
		}
	}
	return it, msgs
}

func threadGenerator(t *testing.T, parentTS string, num int, chunkSz int) (iter.Seq2[slack.Message, error], [][]slack.Message) {
	start, err := fasttime.TS2int(parentTS)
	if err != nil {
		t.Fatalf("failed to convert timestamp: %v", err)
	}
	// generating messages
	msg := make([]slack.Message, num)
	for i := range num {
		ts := fasttime.Int2TS(start + int64(i))
		m := slack.Message{
			Msg: slack.Msg{
				Timestamp:       ts,
				Text:            "thread msg: " + ts,
				ThreadTimestamp: parentTS,
			},
		}
		msg[i] = m
	}
	msgs := slices.Collect(slices.Chunk(msg, chunkSz))
	// creating iterator
	it := func(yield func(slack.Message, error) bool) {
		for _, m := range msg {
			if !yield(m, nil) {
				break
			}
		}
	}
	return it, msgs
}

func Test_encodeThreadMessages(t *testing.T) {
	parentMsg := &slack.Message{Msg: slack.Msg{Timestamp: "123.456", ThreadTimestamp: "123.456"}}
	type args struct {
		ctx context.Context
		// rec      processor.Conversations
		// src      source.Sourcer
		ch       *slack.Channel
		par      *slack.Message
		threadTS string
	}
	tests := []struct {
		name     string
		args     args
		expectFn func(t *testing.T, mc *mock_processor.MockConversations, ms *mock_source.MockSourcer)
		wantErr  bool
	}{
		{
			name: "ok",
			args: args{
				ctx:      t.Context(),
				ch:       structures.ChannelFromID("C123"),
				par:      parentMsg,
				threadTS: "123.456",
			},
			expectFn: func(t *testing.T, mc *mock_processor.MockConversations, ms *mock_source.MockSourcer) {
				it, msgs := threadGenerator(t, "123.456", 203, defaultChunkSize)
				ms.EXPECT().AllThreadMessages(gomock.Any(), "C123", "123.456").Return(it, nil)
				for i := range len(msgs) - 1 { // this should be called 203/100=2 times
					mc.EXPECT().ThreadMessages(gomock.Any(), "C123", *parentMsg, false, false, msgs[i]).Return(nil)
				}
				// last flush call, called with the 3 message chunk
				mc.EXPECT().ThreadMessages(gomock.Any(), "C123", *parentMsg, false, true, msgs[len(msgs)-1]).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mc := mock_processor.NewMockConversations(ctrl)
			ms := mock_source.NewMockSourcer(ctrl)
			if tt.expectFn != nil {
				tt.expectFn(t, mc, ms)
			}
			if err := encodeThreadMessages(tt.args.ctx, mc, ms, tt.args.ch, tt.args.par, tt.args.threadTS); (err != nil) != tt.wantErr {
				t.Errorf("encodeThreadMessages() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
