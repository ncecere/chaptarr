package download

import (
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

func TestInfoHash(t *testing.T) {
	// "info" appears as a string value before the real key, and the info
	// dictionary nests a list of dictionaries.
	const info = "d5:filesld6:lengthi1e4:pathl1:aeee4:name1:xe"
	torrent := "d7:comment4:info4:info" + info + "8:url-listl1:uee"
	got, err := torrentInfoHash([]byte(torrent))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha1.Sum([]byte(info))
	if want := hex.EncodeToString(sum[:]); got != want {
		t.Fatalf("hash %q, want %q", got, want)
	}
	for _, bad := range []string{"", "le", "d4:infod", "d4:info", "d3:fooi1ee", "<html>login</html>"} {
		if _, err := torrentInfoHash([]byte(bad)); err == nil {
			t.Errorf("torrentInfoHash(%q) should fail", bad)
		}
	}
}

func TestMagnetInfoHash(t *testing.T) {
	const want = "c12fe1c06bba254a9dc9f519b335aa7c1367a88a"
	cases := map[string]string{
		"magnet:?xt=urn:btih:" + want + "&dn=x":                             want,
		"magnet:?dn=x&xt=urn:btih:C12FE1C06BBA254A9DC9F519B335AA7C1367A88A": want,
		"magnet:?xt=urn:btih:YEX6DQDLXISUVHOJ6UM3GNNKPQJWPKEK":              want,
	}
	for link, want := range cases {
		if got, err := magnetInfoHash(link); err != nil || got != want {
			t.Errorf("magnetInfoHash(%q) = %q, %v; want %q", link, got, err, want)
		}
	}
	if _, err := magnetInfoHash("magnet:?xt=urn:btmh:1220abcd"); err == nil {
		t.Error("a v2-only magnet should be rejected")
	}
}
