package fs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/trillian/experimental/logless/api"
)

// FS is a logless storage implementation which uses files to store tree state.
// The on-disk structure is:
//  <rootDir>/leaves/aa/bb/cc/ddeeff...
//  <rootDir>/leaves/pending/
//  <rootDir>/seq/aa/bb/cc/ddeeff...
//  <rootDir>/tile/<level>/aa/bb/ccddee...
//  <rootDir>/state
type FS struct {
	rootDir string
	nextSeq uint64
	state   api.LogState
}

const (
	leavesPendingPathFmt = "leaves/pending/%0x"
	statePath            = "state"
)

// New returns an FS instance initialised from the filesystem.
func New(rootDir string) (*FS, error) {
	fi, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat %q: %w", rootDir, err)
	}

	if !fi.IsDir() {
		return nil, fmt.Errorf("%q is not a directory", rootDir)
	}

	s, err := loadLogState(filepath.Join(rootDir, statePath))
	if err != nil {
		return nil, err
	}

	return &FS{
		rootDir: rootDir,
		state:   *s,
		nextSeq: s.Size,
	}, nil
}

// Create creates a new filesystem hierarchy and returns an FS representation for it.
func Create(rootDir string, emptyHash []byte) (*FS, error) {
	_, err := os.Stat(rootDir)
	if err == nil {
		return nil, fmt.Errorf("%q already exists", rootDir)
	}

	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %q: %w", rootDir, err)
	}

	for _, sfx := range []string{"leaves/pending", "seq", "tree"} {
		path := filepath.Join(rootDir, sfx)
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %q: %w", path, err)
		}
	}

	fs := &FS{
		rootDir: rootDir,
	}

	logState := api.LogState{
		Size:     0,
		RootHash: emptyHash,
		Hashes:   [][]byte{},
	}

	if err := fs.UpdateState(logState); err != nil {
		return nil, err
	}
	return fs, nil
}

// LogState returns the current LogState.
func (fs *FS) LogState() api.LogState {
	return fs.state
}

// UpdateState updates the stored log state.
func (fs *FS) UpdateState(newState api.LogState) error {
	fs.state = newState
	fs.nextSeq = newState.Size
	lsRaw, err := json.Marshal(newState)
	if err != nil {
		return fmt.Errorf("failed to marshal LogState: %w", err)
	}
	return ioutil.WriteFile(filepath.Join(fs.rootDir, statePath), lsRaw, 0644)
}

func seqPath(root string, seq uint64) (string, string) {
	frag := []string{
		root,
		"seq",
		fmt.Sprintf("%02x", (seq >> 32)),
		fmt.Sprintf("%02x", (seq>>24)&0xff),
		fmt.Sprintf("%02x", (seq>>16)&0xff),
		fmt.Sprintf("%02x", (seq>>8)&0xff),
		fmt.Sprintf("%02x", seq&0xff),
	}
	d := filepath.Join(frag[:6]...)
	return d, frag[6]
}

func leafPath(root string, leafhash []byte) (string, string) {
	frag := []string{
		root,
		"leaves",
		fmt.Sprintf("%02x", leafhash[0]),
		fmt.Sprintf("%02x", leafhash[1]),
		fmt.Sprintf("%02x", leafhash[2]),
		fmt.Sprintf("%0x", leafhash[3:]),
	}
	d := filepath.Join(frag[:5]...)
	return d, frag[5]
}

func tilePath(root string, level, index uint64) (string, string) {
	frag := []string{
		root,
		"tile",
		fmt.Sprintf("%02x", level),
		fmt.Sprintf("%0x", (index >> 24)),
		fmt.Sprintf("%02x", (index>>16)&0xff),
		fmt.Sprintf("%02x", (index>>8)&0xff),
		fmt.Sprintf("%02x", index&0xff),
	}
	d := filepath.Join(frag[:6]...)
	return d, frag[6]
}

// Sequence assigns the given leaf entry to the next available sequence number.
func (fs *FS) Sequence(leafhash []byte, leaf []byte) error {
	// First store the entry in a temp file
	tmp := filepath.Join(fs.rootDir, fmt.Sprintf(leavesPendingPathFmt, leafhash))
	if err := ioutil.WriteFile(tmp, leaf, 0644); err != nil {
		return fmt.Errorf("unable to write leafdata to temporary file: %w", err)
	}
	defer func() {
		os.Remove(tmp)
	}()

	// Try to link into leaf data storage
	leafDir, leafFile := leafPath(fs.rootDir, leafhash)
	if err := os.MkdirAll(leafDir, 0755); err != nil {
		return fmt.Errorf("failed to make leaf directory structure: %w", err)
	}
	if err := os.Link(tmp, filepath.Join(leafDir, leafFile)); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to link leafdata file: %w", err)
	}

	// Now try to sequence it
	for {
		seq := fs.nextSeq
		fs.nextSeq++

		seqDir, seqFile := seqPath(fs.rootDir, seq)
		if err := os.MkdirAll(seqDir, 0755); err != nil {
			return fmt.Errorf("failed to make seq directory structure: %w", err)
		}
		if err := os.Link(tmp, filepath.Join(seqDir, seqFile)); err != nil {
			continue
		}
		break
	}

	return nil
}

// ScanSequenced calls the provided function once for each contiguous entry
// in storage starting at begin.
// The scan will abort if the function returns an error.
func (fs *FS) ScanSequenced(begin uint64, f func(seq uint64, entry []byte) error) error {
	for {
		sp := filepath.Join(seqPath(fs.rootDir, begin))
		entry, err := ioutil.ReadFile(sp)
		if os.IsNotExist(err) {
			// we're done.
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read leafdata at index %d: %w", begin, err)
		}
		if err := f(begin, entry); err != nil {
			return err
		}
		begin++
	}
}

// GetTile returns the tile at the given tile-level and tile-index.
// If no complete tile exists at that location, it will attempt to find a
// partial tile for the given tree size at that location.
func (fs *FS) GetTile(level, index, logsize uint64) (*api.Tile, error) {
	p := filepath.Join(tilePath(fs.rootDir, level, index))
	t, err := ioutil.ReadFile(p)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read tile at %q: %w", p, err)
	}
	if err == nil {
		return parseTile(t)
	}
	// Not found, so try a fractional tile
	p += fmt.Sprintf(".%02x", logsize%0xff)
	t, err = ioutil.ReadFile(p)
	if err != nil {
		// Do not wrap this error - caller needs to know if it's IsNotExists.
		return nil, err
	}
	return parseTile(t)
}

func parseTile(t []byte) (*api.Tile, error) {
	var tile api.Tile
	if err := json.Unmarshal(t, &tile); err != nil {
		return nil, fmt.Errorf("failed to parse tile: %w", err)
	}
	return &tile, nil
}

// StoreTile writes a tile out to disk.
func (fs *FS) StoreTile(level, index, tileSize uint64, tile *api.Tile) error {
	if tileSize > 256 || tileSize == 0 {
		return fmt.Errorf("tileSize %d must be > 0 and <= 256", tileSize)
	}
	t, err := json.Marshal(tile)
	if err != nil {
		return fmt.Errorf("failed to marshal tile: %w", err)
	}

	tDir, tFile := tilePath(fs.rootDir, level, index)
	tPath := filepath.Join(tDir, tFile)

	if err := os.MkdirAll(tDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %q: %w", tDir, err)
	}

	if tileSize != 256 {
		tPath += fmt.Sprintf(".%02x", tileSize)
	}
	// TODO(al): use unlinked temp file
	temp := fmt.Sprintf("%s.temp", tPath)
	if err := ioutil.WriteFile(temp, t, 0644); err != nil {
		return fmt.Errorf("failed to write temporary tile file: %w", err)
	}
	if err := os.Rename(temp, tPath); err != nil {
		return fmt.Errorf("failed to rename temporary tile file: %w", err)
	}

	if tileSize == 256 {
		// TODO(al): Attempt to clean up old partial tiles by making them be links to the full tile.
	}

	return nil
}

func loadLogState(s string) (*api.LogState, error) {
	raw, err := ioutil.ReadFile(s)
	if err != nil {
		return nil, err
	}

	var ls api.LogState
	if err := json.Unmarshal(raw, &ls); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logstate: %w", err)
	}
	return &ls, nil
}
