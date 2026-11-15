package download

import (
	"crypto/sha1"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
)

// torrentInfoHash returns the hex SHA-1 of the bencoded "info" dictionary,
// which is how torrent clients identify a torrent.
func torrentInfoHash(data []byte) (string, error) {
	if len(data) == 0 || data[0] != 'd' {
		return "", errors.New("not a bencoded dictionary")
	}
	pos := 1
	for pos < len(data) && data[pos] != 'e' {
		key, next, err := bencodeString(data, pos)
		if err != nil {
			return "", err
		}
		end, err := bencodeSkip(data, next, 0)
		if err != nil {
			return "", err
		}
		if key == "info" {
			sum := sha1.Sum(data[next:end])
			return hex.EncodeToString(sum[:]), nil
		}
		pos = end
	}
	return "", errors.New("no info dictionary")
}

func bencodeString(data []byte, pos int) (string, int, error) {
	n := 0
	i := pos
	for ; i < len(data) && data[i] >= '0' && data[i] <= '9'; i++ {
		n = n*10 + int(data[i]-'0')
		if n > len(data) {
			return "", 0, errors.New("bad string length")
		}
	}
	if i == pos || i >= len(data) || data[i] != ':' || i+1+n > len(data) {
		return "", 0, errors.New("malformed string")
	}
	return string(data[i+1 : i+1+n]), i + 1 + n, nil
}

// bencodeSkip returns the offset just past the value starting at pos.
func bencodeSkip(data []byte, pos, depth int) (int, error) {
	if pos >= len(data) || depth > 100 {
		return 0, errors.New("truncated or too deeply nested")
	}
	switch c := data[pos]; {
	case c == 'i':
		end := pos + 1
		for end < len(data) && data[end] != 'e' {
			end++
		}
		if end >= len(data) {
			return 0, errors.New("unterminated integer")
		}
		return end + 1, nil
	case c == 'l' || c == 'd':
		pos++
		for pos < len(data) && data[pos] != 'e' {
			next, err := bencodeSkip(data, pos, depth+1)
			if err != nil {
				return 0, err
			}
			pos = next
		}
		if pos >= len(data) {
			return 0, errors.New("unterminated list or dictionary")
		}
		return pos + 1, nil
	case c >= '0' && c <= '9':
		_, next, err := bencodeString(data, pos)
		return next, err
	}
	return 0, errors.New("unexpected byte")
}

// magnetInfoHash extracts the v1 info-hash from a magnet link as lowercase
// hex. Magnets may carry it in hex or base32.
func magnetInfoHash(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", errors.New("invalid magnet link")
	}
	for _, xt := range u.Query()["xt"] {
		h, ok := strings.CutPrefix(xt, "urn:btih:")
		if !ok {
			continue
		}
		switch len(h) {
		case 40:
			if _, err := hex.DecodeString(h); err == nil {
				return strings.ToLower(h), nil
			}
		case 32:
			if raw, err := base32.StdEncoding.DecodeString(strings.ToUpper(h)); err == nil {
				return hex.EncodeToString(raw), nil
			}
		}
	}
	return "", errors.New("magnet link has no BitTorrent v1 info-hash")
}
