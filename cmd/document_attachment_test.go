package cmd

import "testing"

// attach_seq は添付ファイル一覧取得が返す seq をそのまま渡す値で、0 になり得る
// （1 件目の添付ファイルの seq が 0 の書類を実際に確認した）。0 を拒否すると
// その添付ファイルを取得・更新・削除できない。
func TestParseAttachSeq(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"0", 0, false},
		{"1", 1, false},
		{"  2  ", 2, false},
		{"-1", 0, true},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		got, err := parseAttachSeq(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseAttachSeq(%q) = %d, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseAttachSeq(%q) err = %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("parseAttachSeq(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}
