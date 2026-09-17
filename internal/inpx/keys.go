package inpx

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/alexbelweb/flibustahub/internal/textnorm"
)

// DisplayName is "Last First Middle" with collapsed inner spaces.
func (a Author) DisplayName() string {
	parts := make([]string, 0, 3)
	for _, p := range []string{a.Last, a.First, a.Middle} {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, " ")
}

// Key is normalize(last)+","+normalize(first)+","+normalize(middle).
func (a Author) Key() string {
	return textnorm.Normalize(a.Last) + "," + textnorm.Normalize(a.First) + "," + textnorm.Normalize(a.Middle)
}

// SortName is the denormalized author ordering key.
func (a Author) SortName() string {
	return strings.TrimSpace(textnorm.Normalize(a.Last) + " " + textnorm.Normalize(a.First) + " " + textnorm.Normalize(a.Middle))
}

// AuthorsText joins display names in AUTHOR-field order.
func AuthorsText(authors []Author) string {
	names := make([]string, len(authors))
	for i, a := range authors {
		names[i] = a.DisplayName()
	}
	return strings.Join(names, ", ")
}

// SortTitle is normalize(title).
func SortTitle(title string) string {
	return textnorm.Normalize(title)
}

// WorkKey returns hex(sha256(preimage)) and the preimage (for tests/debug only).
func WorkKey(authors []Author, title string) (key, preimage string) {
	keys := make([]string, len(authors))
	for i, a := range authors {
		keys[i] = a.Key()
	}
	sort.Strings(keys)
	preimage = strings.Join(keys, "|") + "\t" + textnorm.Normalize(title)
	sum := sha256.Sum256([]byte(preimage))
	return hex.EncodeToString(sum[:]), preimage
}

var recencyRe = regexp.MustCompile(`(\d+)(?:\.(?:zip|inp))?$`)

// ArchiveRecency is the number immediately before .zip in an archive file name.
func ArchiveRecency(name string) int {
	base := name
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	m := recencyRe.FindStringSubmatch(strings.ToLower(base))
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return n
}

// EditionState is the stored edition fields shouldReplace needs.
type EditionState struct {
	IsDeleted   bool
	ArchiveName string
}

// ShouldReplace decides whether an incoming record overwrites a stored edition.
func ShouldReplace(existing EditionState, incoming Record) bool {
	if existing.IsDeleted && !incoming.IsDeleted {
		return true
	}
	if !existing.IsDeleted && incoming.IsDeleted {
		return false
	}
	return ArchiveRecency(incoming.ArchiveName) >= ArchiveRecency(existing.ArchiveName)
}
