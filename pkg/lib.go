package rtimg

import (
	"fmt"
	// "os"
	"path/filepath"
	"regexp"
	"strings"
)

type (
	ITagname interface {
		GetTag(string) (string, error)
		Source() string
		State() error
	}
	TKey struct {
		// if name is empty it must be calculated as a preceding segment of a <key>
		name string
		// filename's size tag. ("800x600" for example)
		size string
		// index starting from last segment
		level    int
		segments []string
	}
	TKeyData struct {
		Type          string
		FileSizeLimit int64
	}
)

const (
	none = -1
	kb   = 1000
	mb   = kb * 1000
	gb   = mb * 1000
)

var validExtension = map[string]bool{}

var cannotBeProjectName = []string{
	"./для сервиса/",
}

var doOffsetForProjectName = []*regexp.Regexp{
	regexp.MustCompile(`^\d+ сезон$`),
}

var postersTable = map[string]TKeyData{
	"./350x500.jpg":  {"rt", 1 * mb},
	"./350x500.psd":  {"rt", none},
	"./525x300.jpg":  {"rt", 1 * mb},
	"./525x300.psd":  {"rt", none},
	"./810x498.jpg":  {"rt", 1 * mb},
	"./810x498.psd":  {"rt", none},
	"./270x390.jpg":  {"rt", 1 * mb},
	"./270x390.psd":  {"rt", none},
	"./1620x996.jpg": {"rt", 1 * mb},
	"./1620x996.psd": {"rt", none},
	"./503x726.jpg":  {"rt", 1 * mb},
	"./503x726.psd":  {"rt", none},
	"./logo.png":     {"rt", 1 * mb},
	"./logo.psd":     {"rt", none},

	"./600x600.jpg":          {"gp", 700 * kb},
	"./600x600.psd":          {"gp", none},
	"./600x840.jpg":          {"gp", 700 * kb},
	"./600x840.psd":          {"gp", none},
	"./1920x1080.jpg":        {"gp", 700 * kb},
	"./1920x1080.psd":        {"gp", none},
	"./1920x1080_left.jpg":   {"gp", 700 * kb},
	"./1920x1080_left.psd":   {"gp", none},
	"./1920x1080_center.jpg": {"gp", 700 * kb},
	"./1920x1080_center.psd": {"gp", none},
	"./1260x400.jpg":         {"gp", 700 * kb},
	"./1260x400.psd":         {"gp", none},
	"./1080x540.jpg":         {"gp", 700 * kb},
	"./1080x540.psd":         {"gp", none},
	// megafon
	"./1080x810.png":  {"gp", 6 * mb},
	"./1080x810.psd":  {"gp", none},
	"./1080x1232.png": {"gp", 6 * mb},
	"./1080x1232.psd": {"gp", none},
	"./1104x624.png":  {"gp", 6 * mb},
	"./1104x624.psd":  {"gp", none},
	"./3840x1344.png": {"gp", 6 * mb},
	"./3840x1344.psd": {"gp", none},

	// viasat
	"./для сервиса/600x600.jpg":   {"gp", 3 * mb},
	"./для сервиса/600x600.psd":   {"gp", none},
	"./для сервиса/600x840.jpg":   {"gp", 3 * mb},
	"./для сервиса/600x840.psd":   {"gp", none},
	"./для сервиса/1920x1080.jpg": {"gp", 3 * mb},
	"./для сервиса/1920x1080.psd": {"gp", none},
	"./для сервиса/1760x557.jpg":  {"gp", 3 * mb},
	"./для сервиса/1760x557.psd":  {"gp", none},

	"./google_apple_feed/jpg/g_hasLogo_600x600.png":         {"gp", none},
	"./google_apple_feed/psd/g_hasLogo_600x600.psd":         {"gp", none},
	"./google_apple_feed/jpg/g_hasTitle_logo_1800x1000.png": {"gp", none},
	"./google_apple_feed/psd/g_hasTitle_logo_1800x1000.psd": {"gp", none},

	"./google_apple_feed/jpg/g_iconic_poster_600x600.jpg":       {"gp", 3 * mb},
	"./google_apple_feed/psd/g_iconic_poster_600x600.psd":       {"gp", none},
	"./google_apple_feed/jpg/g_iconic_poster_600x800.jpg":       {"gp", 3 * mb},
	"./google_apple_feed/psd/g_iconic_poster_600x800.psd":       {"gp", none},
	"./google_apple_feed/jpg/g_iconic_poster_800x600.jpg":       {"gp", 3 * mb},
	"./google_apple_feed/psd/g_iconic_poster_800x600.psd":       {"gp", none},
	"./google_apple_feed/jpg/g_iconic_poster_1000x1500.jpg":     {"gp", 3 * mb},
	"./google_apple_feed/psd/g_iconic_poster_1000x1500.psd":     {"gp", none},
	"./google_apple_feed/jpg/g_iconic_poster_3840x2160.jpg":     {"gp", 3 * mb},
	"./google_apple_feed/psd/g_iconic_poster_3840x2160.psd":     {"gp", none},
	"./google_apple_feed/jpg/g_iconic_background_1000x1500.jpg": {"gp", 3 * mb},
	"./google_apple_feed/psd/g_iconic_background_1000x1500.psd": {"gp", none},
	"./google_apple_feed/jpg/g_iconic_background_3840x2160.jpg": {"gp", 3 * mb},
	"./google_apple_feed/psd/g_iconic_background_3840x2160.psd": {"gp", none},
}

var reSize = regexp.MustCompile(`^(?:.*_)?(?:(\d+x\d+)|(logo))[\._].*$`)

func init() {
	// gather valid extensions
	for v, _ := range postersTable {
		ext := filepath.Ext(v)
		validExtension[ext] = true
	}
	// fmt.Printf("debug: valid extensions: %v\n", validExtension)
}

func CheckImage(filePath string, tn ITagname) (*TKeyData, error) {
	key, err := FindKey(filePath, tn)
	if err != nil {
		return nil, err
	}
	data := key.Data()
	if data == nil {
		return nil, fmt.Errorf("unreachable: something wrong with a <key>")
	}
	return data, nil
}

func GetProjectDir(filePath string) string {
	key, _ := FindKey(filePath, nil)
	if key == nil {
		return ""
	}
	return key.ProjectDir()
}

// FindKey - tagname will be used only if it failed to find <key> for the path
func FindKey(path string, tn ITagname) (*TKey, error) {
	name := ""
	key, err := tryToFindKey(path, name)
	if err != nil {
		if tn == nil {
			return nil, fmt.Errorf("findKey: tagname is <nil>")
		}
		// check if the pointer over interface is nil
		if tn.State() != nil {
			return nil, fmt.Errorf("findKey: <key> not found")
		}
		path, err = pathFromTagname(tn)
		if err != nil {
			return nil, fmt.Errorf("findKey: %v", err)
		}
		name, err = makeNameUsingTags(tn)
		if err != nil {
			return nil, fmt.Errorf("findKey: %v", err)
		}
		key, err = tryToFindKey(path, name)
		if err != nil {
			return nil, fmt.Errorf("findKey: %v", err)
		}
	}
	return key, nil
}

func newKey(path string, name string) (*TKey, error) {
	p := filepath.Clean(path)
	// ### TODO ###: seems to be a dirty hack
	p = strings.ReplaceAll(p, "\\", "/")

	segments := strings.Split(p, "/")

	list := reSize.FindAllString(segments[len(segments)-1], -1)
	if len(list) != 1 {
		return nil, fmt.Errorf("newKey: something wrong with a size tag")
	}
	size := list[0]
	return &TKey{segments: segments, name: name, size: size}, nil
}

func (o *TKey) Len() int {
	return len(o.segments)
}

func (o *TKey) Segment(n int) (string, bool) {
	if n < 0 || n >= len(o.segments) {
		return "", false
	}
	return o.segments[n], true
}

func (o *TKey) Hash() string {
	if o.level < 0 || o.level >= len(o.segments) {
		return ""
	}
	idx := len(o.segments) - 1 - o.level
	ret := strings.Join(o.segments[idx:], "/")
	// fmt.Println("debug: ", ret)
	return "./" + ret
}

func (o *TKey) NextLevel() bool {
	if o.Hash() == "" {
		return false
	}
	o.level++
	return true
}

func (o *TKey) Name() string {
	if o.name != "" {
		return o.name
	}
	ret, _ := o.Segment(len(o.segments) - 2 - o.level)
	// fmt.Printf("\ndebug: name: %q level %v segs: %v\n", o.name, o.level, strings.Join(o.segments, "/"))
	return ret
}

func (o *TKey) Data() *TKeyData {
	hash := o.Hash()
	fmt.Printf("hash: %v\n", hash)
	/*
		path := strings.TrimPrefix(hash, "./")

		for _, prefix := range cannotBeProjectName {
			h := prefix + path
			if _, ok := postersTable[h]; ok {
				return nil
			}
		}
	*/

	if ret, ok := postersTable[hash]; ok {
		return &ret
	}

	return nil
}

func (o *TKey) Size() string {
	return o.size
}

func (o *TKey) Base() string {
	return o.segments[len(o.segments)-1]
}

func (o *TKey) ProjectDir() string {
	idx := len(o.segments) - 1 - o.level
	if idx < 1 {
		return ""
	}
	return strings.Join(o.segments[:idx], "/")
}

func (o *TKey) String() string {
	if o == nil {
		return fmt.Sprintf("%v", nil)
	}
	return fmt.Sprintf("name: %v, size: %v, level: %v, segments: %v", o.name, o.size, o.level, o.segments)
}

func makeNameUsingTags(tn ITagname) (string, error) {
	nameTags := []string{"name", "sxx", "sname", "exx", "ename", "comment", "year", "sdhd"}
	name := []string{}
	for _, tag := range nameTags {
		val, _ := tn.GetTag(tag)
		if val == "" {
			continue
		}
		name = append(name, val)
	}
	if len(name) == 0 {
		return "", fmt.Errorf("%v does not have enough tags to construct <name>", tn.Source())
	}
	return strings.Join(name, "_"), nil
}

func pathFromTagname(tn ITagname) (string, error) {
	if tn == nil {
		return "", fmt.Errorf("make <key>: tagname is <nil>")
	}

	base, err := tn.GetTag("sizetag")
	if err != nil {
		return "", fmt.Errorf("make <key>: %v", err)
	}
	align, _ := tn.GetTag("aligntag")
	if align != "" {
		base += "_" + align
	}
	base += filepath.Ext(tn.Source())
	ret := filepath.Join(filepath.Dir(tn.Source()), base)
	return ret, nil
}

func tryToFindKey(path string, name string) (*TKey, error) {
	key, err := newKey(path, name)
	if err != nil {
		return nil, err
	}
	for {
		fmt.Printf("debug: %v, %v\n", key.Hash(), key.Data())
		if key.Data() != nil {
			return key, nil
		}
		if key.NextLevel() {
			continue
		}
		return nil, fmt.Errorf("tryToFindKey(): <key> not found %v", key)
	}
}
