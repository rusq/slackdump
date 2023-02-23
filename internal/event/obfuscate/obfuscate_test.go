package obfuscate

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"hash"
	"io"
	"math/rand"
	"testing"

	"github.com/rusq/slackdump/v2/internal/event"
	"github.com/rusq/slackdump/v2/internal/fixtures"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(0) // make it deterministic
}

func Test_randomString(t *testing.T) {
	type args struct {
		n int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty",
			args: args{n: 0},
			want: "jXUJR9JT5pul5g8MDbK7E1ycTwBhzdJG9 ",
		},
		{
			name: "one",
			args: args{n: 1},
			want: "VwGabEN7FkWNmyD0HtOdvcYYvfHfF hVA6",
		},
		{
			name: "100",
			args: args{n: 100},
			want: "d1BtVOw52BH40tQ4xsZr1rbOEdndtLrooKH5L9GzLgWmmWfVTBKfSvym98qEQMYaWdLEKrJCEXzYB2bFiOLzhKfgf0hdxneHm6GIP4BlU7M3cWoFQL4mevBBbRf",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := randomString(tt.args.n); got != tt.want {
				t.Errorf("randomString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_Do(t *testing.T) {
	var buf bytes.Buffer
	src := fixtures.EventsJSONL()
	if err := Do(context.Background(), &buf, src); err != nil {
		t.Fatal(err)
	}
	// reopen
	src.Close()
	src = fixtures.EventsJSONL()
	srcEvt := unmarshalEvents(src)
	dstEvt := unmarshalEvents(&buf)
	if len(srcEvt) != len(dstEvt) {
		t.Fatalf("expected %d events, got %d", len(srcEvt), len(dstEvt))
	}
	// ensure that text is obfuscated.
	for i := range srcEvt {
		if srcEvt[i].Type != dstEvt[i].Type {
			t.Fatalf("expected %q, got %q", srcEvt[i].Type, dstEvt[i].Type)
		}
		if srcEvt[i].Type == event.EMessages {
			for j := range srcEvt[i].Messages {
				if srcEvt[i].Messages[j].Text == dstEvt[i].Messages[j].Text && srcEvt[i].Messages[j].Text != "" {
					t.Fatalf("expected %q, got %q", srcEvt[i].Messages[j].Text, dstEvt[i].Messages[j].Text)
				}
			}
		}
	}
}

func unmarshalEvents(r io.Reader) []event.Event {
	var events []event.Event
	dec := json.NewDecoder(r)
	for {
		var e event.Event
		if err := dec.Decode(&e); err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		events = append(events, e)
	}
	return events
}

func Test_obfuscator_OneMessage(t *testing.T) {
	rand.Seed(0)
	type fields struct {
		hasher func() hash.Hash
		salt   string
	}
	type args struct {
		m *slack.Message
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantMsg *slack.Message
	}{
		{
			name: "empty",
			fields: fields{
				hasher: sha1.New,
				salt:   "salt",
			},
			args: args{
				m: &slack.Message{},
			},
			wantMsg: &slack.Message{},
		},
		{
			name: "text",
			fields: fields{
				hasher: sha1.New,
				salt:   "salt",
			},
			args: args{
				m: fixtures.Load[*slack.Message](fixtures.SimpleMessageJSON),
			},
			wantMsg: &slack.Message{
				Msg: slack.Msg{
					ClientMsgID: "a29ab0f5-808b-bc8e-f22e-b4ac1a00fcd4",
					Type:        "message",
					Channel:     "",
					User:        "U8EEA06E1",
					Text:        "9 LVwGabEN7FkWNmyD0HtOdvcYYvfHfF hVA6Nd1BtVOw52BH40tQ4xsZr1rbOE",
					Timestamp:   "1645095505.023899",
					Team:        "TEBC93378",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := obfuscator{
				hasher: tt.fields.hasher,
				salt:   tt.fields.salt,
			}
			o.OneMessage(tt.args.m)
			if !assert.Equal(t, tt.wantMsg, tt.args.m) {
				t.Errorf("obfuscator.OneMessage() = %v, want %v", tt.args.m, tt.wantMsg)
			}
		})
	}
}

func Test_obfuscator_OneFile(t *testing.T) {
	rand.Seed(0)
	type fields struct {
		hasher func() hash.Hash
		salt   string
	}
	type args struct {
		f *slack.File
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantFile *slack.File
	}{
		{
			name: "empty",
			fields: fields{
				hasher: sha1.New,
				salt:   "salt",
			},
			args: args{
				f: &slack.File{},
			},
			wantFile: &slack.File{},
		},
		{
			name: "test file",
			fields: fields{
				hasher: sha1.New,
				salt:   "salt",
			},
			args: args{
				f: fixtures.Load[*slack.File](fixtures.FileJPEG),
			},
			wantFile: &slack.File{
				ID:                 "F8B5BAA15C4",
				Created:            1638784624,
				Timestamp:          1638784624,
				Name:               "N1CIe93m6sjyQtxxQ",
				Title:              "NrZ60 cPGC7",
				Mimetype:           "image/jpeg",
				Filetype:           "jpg",
				PrettyType:         "JPEG",
				User:               "U8EEA06E1",
				Mode:               "hosted",
				Size:               359002,
				URLPrivate:         "https://files.slack.com/jXUJR9JT5pul5g8MDbK7E1ycTwBhzdJG9 LVwGabEN7FkWNmyD0HtOdvcYYvfHfF hVA6Nd1BtVOw52BH40tQ4xsZr1rbOEdndtLrooKH5L9GzLgWmmWfVTBKfSvym98qEQMYaWdLEKrJCEXzYB2bFiOLzhK",
				URLPrivateDownload: "https://files.slack.com/gf0hdxneHm6GIP4BlU7M3cWoFQL4mevBBbRfBaJPco41JqcXJevtl3KAQUasyhcDmjIACMVY8RiwNtUgvZE2pjKqhFdMiy4OxOLXgpn9vgcFkrDxZeggmJqKQer831r0R4HiQsusLuMAyAFJbjLBxTnKG74XmDCtXQ5z",
				Thumb64:            "https://files.slack.com/ohUTYVSO55A6moITgdTl0IcFo3pj50 bDZ55HpiAhMyxNa9ZJKd K2 5leFM7x1YbtBppBXxH7QllTEzOIJuF 2a3JuSmPeBPcVVkBgSgaDkz1gBZPPbFyitYFB5KJwfCZEdQ9VtrchKQPXsP",
				Thumb80:            "https://files.slack.com/X8pShmaP8wAqT0bfFZpEFWl3O76Z 6nPjIUopX0QpfW2l9co9gRSs0LbUwL1T7Q2CDqGDgS7kw9guK2H3Ojlr323ucxm2ILBEEqGMj8MEji4HAH20RLD3 SRwQwJG4PFKxtMVDPEyanTRbFE2kb1dlg0qwRa5",
				Thumb160:           "https://files.slack.com/E6DfA23da6kZGkDGx3QrRiUPwLDhH85x2Anr0qPPs 37 KKwBxkYoVZsSD7PDMLOPP0 ImN86qfEhi 8YW94ufGAfx6dSud0oWWLRDPOINd34kmmj0ZRO3F4OUiOHo3MGMhEyp2igPvErNr4vZaOi9ZXtN",
				Thumb360:           "https://files.slack.com/0K3rhJUvtLkbqckRUiDgTZvPDfpK8wZ0DhSBY2E9pwF7n7qHV8TA23OJak5BSN2n9eExD9wVZDNW2Fj33R3WLTDMvyNZZl48Pp3SpNs0vVkybaC2wTpwoZwmC6HYLn6NvjUWiHe0yJpeBCZgENc",
				Thumb480:           "https://files.slack.com/UuSjQK0RHcO24AMRHNfX6dFvbHQL73IDO4YnZVlE90G7ci03sGuCkwQV7U2JRHAumduWXi3mLEUA5XbovxT46p8h2nWyoxvlnwFqsrYhC4jR ttzGJyhps6AJVHhDaphPd 152WIjgDrso90ma",
				Thumb720:           "https://files.slack.com/YXjWfW2XgJTQOGaG6cwYqDtYsKI6PenZdjIz0izbnNdXDVP23S5Nkr6NLH9IhBpQ0KRPAkkCMVmXEbn26CKWzp9JtsLArhI6EEMpm5Cza3BRE8hi6HreaVlsiA8yWRGFIhf45i1pQDy0TiYAMiv9eBQQK70kSF1",
				Thumb960:           "https://files.slack.com/icJt4QcFmcr6FbZ6bwr7ohT4y829DAd7Hp8K9nhq4gqvaibgt6L8KZxRntuu1LLeyt0E89KDLJh yUvN3noSC72m6rKj0WtHcR25fmf yQjAQjb1g KF7GSwLSVOr9OVpeJzNMcG0Scw jQqRXCoJEQpTTQQMuloIxt9yfvz9MbOh",
				Thumb1024:          "https://files.slack.com/rXhBDD1p6w7JKTSEhdTm73lHLOsqaC86P6pRH7a8cuJJwjezUgvq28748gz9LUWadZQUDJimANwgxHETAg0gQTetSLeiPwqjFe8VgGiYFWcviSkWpXTnwuzjIZsp05 yL2DQRB7Z3z6cY6PLK",
				Permalink:          "UD7AHiFDOQ8Z5cnRblPtUVdpwP3CNDqsbRxikuHZA6yRTj7b2TXYlPkrSCx13GslzOYqb6KqXm2m4wU1N",
				PermalinkPublic:    "c4F1MuVCSDb Z2HyHJefmikogJnBnRry8AgLKkzj2AgOG0zitrjs clu7",
				IsPublic:           true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := obfuscator{
				hasher: tt.fields.hasher,
				salt:   tt.fields.salt,
			}
			o.OneFile(tt.args.f)
			if !assert.Equal(t, tt.wantFile, tt.args.f) {
				t.Errorf("obfuscator.OneFile() = %v, want %v", tt.args.f, tt.wantFile)
			}
		})
	}
}