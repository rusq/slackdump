package renderer

import "testing"

func TestRoutes_RewriteSlackURL(t *testing.T) {
	routes := NewRoutes(ModeLive,
		WithWorkspaceURL("https://example.com"),
		WithLiveHost("localhost:8080"),
	)

	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "channel link",
			src:  "https://example.com/archives/C12341",
			want: "/archives/C12341",
		},
		{
			name: "thread permalink",
			src:  "https://example.com/archives/C12341/p1738580940349469?thread_ts=1737716342.919259&cid=C12341",
			want: "/archives/C12341/1737716342.919259#1738580940.349469",
		},
		{
			name: "channel anchor permalink",
			src:  "https://example.com/archives/C12341/p1738580940349469",
			want: "/archives/C12341#1738580940.349469",
		},
		{
			name: "user link",
			src:  "https://example.com/team/U123",
			want: "/team/U123",
		},
		{
			name: "fallback host replacement",
			src:  "https://example.com/help",
			want: "http://localhost:8080/help",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := routes.RewriteSlackURL(tt.src); got != tt.want {
				t.Fatalf("RewriteSlackURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRoutes_Avatar(t *testing.T) {
	t.Run("live mode", func(t *testing.T) {
		r := NewRoutes(ModeLive)
		if got := r.Avatar("U123", "abc.jpg"); got != "/avatars/U123/abc.jpg" {
			t.Fatalf("Avatar() = %q, want %q", got, "/avatars/U123/abc.jpg")
		}
	})
	t.Run("static mode", func(t *testing.T) {
		r := NewRoutes(ModeStatic)
		if got := r.Avatar("U123", "abc.png"); got != "/avatars/U123/abc.png" {
			t.Fatalf("Avatar() = %q, want %q", got, "/avatars/U123/abc.png")
		}
	})
}

func TestRoutes_StaticPaths(t *testing.T) {
	routes := NewRoutes(ModeStatic)

	if got := routes.Channel("C123"); got != "/archives/C123/index.html" {
		t.Fatalf("Channel() = %q", got)
	}
	if got := routes.Thread("C123", "1710000000.000001"); got != "/archives/C123/threads/1710000000.000001.html" {
		t.Fatalf("Thread() = %q", got)
	}
	if got := routes.Canvas("C123"); got != "/archives/C123/canvas/index.html" {
		t.Fatalf("Canvas() = %q", got)
	}
	if got := routes.CanvasContent("C123"); got != "/archives/C123/canvas/content.html" {
		t.Fatalf("CanvasContent() = %q", got)
	}
	if got := routes.File("F123", "hello world.txt"); got != "/files/F123/hello%20world.txt" {
		t.Fatalf("File() = %q", got)
	}
	if got := routes.File("F123", "a/b:c.txt"); got != "/files/F123/a_b_c.txt" {
		t.Fatalf("File() sanitized = %q", got)
	}
}
