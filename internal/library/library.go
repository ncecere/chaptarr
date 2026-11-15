// Package library handles audiobook files on disk: where a book lives,
// bringing finished downloads in, and finding books already there.
package library

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ncecere/chaptarr/internal/indexer"
	"github.com/ncecere/chaptarr/internal/model"
)

var audioExt = func() map[string]bool {
	m := map[string]bool{}
	for _, f := range indexer.AudioFormats {
		m["."+f] = true
	}
	return m
}()

// Files that travel with the audio: cover art, cue sheets, chapter and
// metadata sidecars.
var sidecarExt = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	".cue": true, ".nfo": true, ".opf": true, ".pdf": true,
}

// IsAudio reports whether the path has an audiobook file extension.
func IsAudio(path string) bool { return audioExt[strings.ToLower(filepath.Ext(path))] }

// Format returns the lowercase extension without the dot.
func Format(path string) string { return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".") }

// cleanSegment makes a string safe as a single path component on every
// common filesystem.
func cleanSegment(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case r < 32:
			return -1
		case r == '/' || r == '\\':
			return '-'
		case r == ':':
			return '-'
		case strings.ContainsRune(`*?"<>|`, r):
			return -1
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	// Windows and SMB shares reject trailing dots and spaces.
	s = strings.TrimRight(s, ". ")
	s = strings.TrimLeft(s, " ")
	if len(s) > 180 {
		s = strings.TrimRight(s[:180], ". ")
	}
	if s == "." || s == ".." {
		return ""
	}
	return s
}

// BookFolder expands the folder template for a book, relative to the root
// folder. Segments that come out empty are dropped, so "{Author}/{Series}/{Title}"
// yields "Author/Title" for a standalone book.
func BookFolder(format string, b *model.Book) string {
	if strings.TrimSpace(format) == "" {
		format = model.DefaultSettings().FolderFormat
	}
	var series, position string
	if b.Series != nil {
		series, position = b.Series.Name, b.Series.Position
	}
	year := ""
	if len(b.ReleaseDate) >= 4 {
		year = b.ReleaseDate[:4]
	}
	narrator := ""
	if len(b.Narrators) > 0 {
		narrator = b.Narrators[0]
	}
	// Token values are cleaned first so a "/" inside a title cannot add a
	// path level.
	rep := strings.NewReplacer(
		"{Author}", cleanSegment(b.AuthorName),
		"{Series}", cleanSegment(series),
		"{SeriesPosition}", cleanSegment(position),
		"{Title}", cleanSegment(b.Title),
		"{Year}", year,
		"{Narrator}", cleanSegment(narrator),
		"{ASIN}", cleanSegment(b.ASIN),
	)
	var parts []string
	for _, seg := range strings.Split(strings.ReplaceAll(format, `\`, "/"), "/") {
		seg = tidy(rep.Replace(seg))
		if seg != "" {
			parts = append(parts, seg)
		}
	}
	if len(parts) == 0 {
		parts = []string{cleanSegment(b.Title)}
		if parts[0] == "" {
			parts[0] = b.ASIN
		}
	}
	return filepath.Join(parts...)
}

// tidy removes the debris an empty token leaves behind, e.g.
// "{SeriesPosition} - {Title}" -> " - Dune" -> "Dune", "() Dune" -> "Dune".
func tidy(seg string) string {
	for _, empty := range []string{"()", "[]", "{}"} {
		seg = strings.ReplaceAll(seg, empty, "")
	}
	seg = strings.Join(strings.Fields(seg), " ")
	seg = strings.Trim(seg, " -_,")
	return cleanSegment(seg)
}

// ---------------------------------------------------------------- import

// Import brings the audio files (and sidecars) under src into destDir and
// returns the audio files now in the library. src may be a directory or a
// single file. With keepSource the source is left intact (required for
// torrents that are still seeding); the mode then decides between hardlink
// and copy. Nothing in the library is overwritten by a different file.
func Import(src, destDir, mode string, keepSource bool) ([]model.BookFile, error) {
	info, err := os.Stat(src)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("download path %q does not exist from Chaptarr's point of view; if the download client runs in a container or on another machine, both must see the files at the same path", src)
		}
		return nil, err
	}

	type entry struct{ abs, rel string }
	var audio, extras []entry
	other := map[string]int{} // extensions of files that are not imported
	if info.IsDir() {
		err = filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(src, p)
			ext := strings.ToLower(filepath.Ext(p))
			switch {
			case strings.HasPrefix(d.Name(), "."), strings.Contains(strings.ToLower(d.Name()), "sample"):
			case audioExt[ext]:
				audio = append(audio, entry{p, rel})
			case sidecarExt[ext]:
				extras = append(extras, entry{p, rel})
				other[ext]++
			default:
				other[ext]++
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("reading download: %w", err)
		}
	} else if IsAudio(src) {
		audio = append(audio, entry{src, filepath.Base(src)})
	}
	if len(audio) == 0 {
		return nil, fmt.Errorf("no audio files found in %q: %s", src, describeContents(other))
	}
	sort.Slice(audio, func(i, j int) bool { return audio[i].rel < audio[j].rel })

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating library folder: %w", err)
	}
	move := mode == "move" && !keepSource
	link := mode == "hardlink" || (mode == "move" && keepSource)

	var out []model.BookFile
	for _, e := range audio {
		dest := filepath.Join(destDir, e.rel)
		size, err := transfer(e.abs, dest, move, link)
		if err != nil {
			return nil, err
		}
		out = append(out, model.BookFile{Path: dest, SizeBytes: size, Format: Format(dest)})
	}
	for _, e := range extras {
		// Sidecars are a nicety; failing on one would strand a good import.
		_, _ = transfer(e.abs, filepath.Join(destDir, e.rel), move, link)
	}
	if move && info.IsDir() {
		removeEmptyDirs(src)
	}
	return out, nil
}

// describeContents says what a download held instead of audio, so a failed
// import explains itself without a shell on the server.
func describeContents(byExt map[string]int) string {
	if len(byExt) == 0 {
		return "the folder is empty from Chaptarr's point of view; check that it is the same folder the download client wrote to, and that Chaptarr's user can read it"
	}
	exts := make([]string, 0, len(byExt))
	archives := false
	for ext := range byExt {
		exts = append(exts, ext)
		if archiveExt[ext] {
			archives = true
		}
	}
	sort.Slice(exts, func(i, j int) bool {
		if byExt[exts[i]] != byExt[exts[j]] {
			return byExt[exts[i]] > byExt[exts[j]]
		}
		return exts[i] < exts[j]
	})
	var parts []string
	for i, ext := range exts {
		if i == 6 {
			parts = append(parts, "…")
			break
		}
		name := ext
		if name == "" {
			name = "no extension"
		}
		parts = append(parts, fmt.Sprintf("%d × %s", byExt[ext], name))
	}
	msg := "it contains " + strings.Join(parts, ", ")
	if archives {
		msg += " (archives are not unpacked)"
	}
	return msg
}

var archiveExt = map[string]bool{".rar": true, ".zip": true, ".7z": true, ".r00": true, ".001": true}

func transfer(src, dest string, move, link bool) (int64, error) {
	si, err := os.Stat(src)
	if err != nil {
		return 0, err
	}
	if di, err := os.Stat(dest); err == nil {
		// Re-running an import (after a crash, or a retry) must be a no-op,
		// but a different file at the same path is not ours to replace.
		if os.SameFile(si, di) || di.Size() == si.Size() {
			return di.Size(), nil
		}
		return 0, fmt.Errorf("%q already exists in the library with different contents", dest)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return 0, err
	}
	if move {
		if err := os.Rename(src, dest); err == nil {
			return si.Size(), nil
		}
		if err := copyFile(src, dest); err != nil {
			return 0, err
		}
		_ = os.Remove(src)
		return si.Size(), nil
	}
	if link {
		if err := os.Link(src, dest); err == nil {
			return si.Size(), nil
		}
		// Different filesystem, or one without hardlinks: fall back to copy.
	}
	if err := copyFile(src, dest); err != nil {
		return 0, err
	}
	return si.Size(), nil
}

// copyFile writes to a temporary name and renames, so an interrupted copy
// never leaves a truncated file that looks like a finished book.
func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dest + ".chaptarr-partial"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(tmp, dest)
	}
	if err != nil {
		os.Remove(tmp)
		return fmt.Errorf("copying %q into the library: %w", filepath.Base(src), err)
	}
	return nil
}

func removeEmptyDirs(root string) {
	var dirs []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			dirs = append(dirs, p)
		}
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Remove(dirs[i]) // only succeeds when empty
	}
}

// DeleteBookFiles removes a book's files, then its folder and any parent
// folders left empty, stopping at (and never removing) a root folder.
func DeleteBookFiles(files []model.BookFile, bookDir string, roots []string) error {
	var firstErr error
	for _, f := range files {
		if err := os.Remove(f.Path); err != nil && !errors.Is(err, fs.ErrNotExist) && firstErr == nil {
			firstErr = err
		}
	}
	if bookDir == "" {
		return firstErr
	}
	var root string
	for _, r := range roots {
		if rel, err := filepath.Rel(r, bookDir); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			root = filepath.Clean(r)
		}
	}
	if root == "" {
		return firstErr
	}
	// Sidecars we imported go with the book; anything else keeps the folder.
	if entries, err := os.ReadDir(bookDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && sidecarExt[strings.ToLower(filepath.Ext(e.Name()))] {
				_ = os.Remove(filepath.Join(bookDir, e.Name()))
			}
		}
	}
	for dir := filepath.Clean(bookDir); dir != root && strings.HasPrefix(dir, root); dir = filepath.Dir(dir) {
		if err := os.Remove(dir); err != nil {
			break
		}
	}
	return firstErr
}

// Exists reports whether the path is present on disk.
func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Mounted reports whether a root folder is present and has content. An
// empty or absent root usually means an unmounted share, in which case
// missing files say nothing about the library.
func Mounted(root string) bool {
	entries, err := os.ReadDir(root)
	return err == nil && len(entries) > 0
}

// Under reports whether path is inside dir.
func Under(dir, path string) bool {
	rel, err := filepath.Rel(dir, path)
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
}
