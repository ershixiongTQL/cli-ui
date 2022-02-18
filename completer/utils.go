package completer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var rawStrBounds = []byte{'"', '\''}

//TODO: Add struct CmdLineFields
type CmdLineFields struct {
	segs []CmdLineSeg
}

func (f *CmdLineFields) Count() int {
	return len(f.segs)
}

func (f *CmdLineFields) Bytes() (bytes [][]byte) {
	for _, s := range f.segs {
		bytes = append(bytes, []byte(s.content))
	}
	return
}

func (f CmdLineFields) Strings() (strs []string) {
	for _, s := range f.segs {
		strs = append(strs, s.content)
	}
	return
}

func (f *CmdLineFields) Append(seg CmdLineSeg) {
	f.segs = append(f.segs, seg)
}

type CmdLineSeg struct {
	content string
	quoted  bool
}

func (s *CmdLineSeg) IsQuoted() bool {
	return s.quoted
}

func (s *CmdLineSeg) UnquotString() string {
	return s.content
}

func (s *CmdLineSeg) String() string {
	if !s.quoted {
		return s.UnquotString()
	} else {
		//TODO:
		return ""
	}
}

//Split a command line into fields
func CmdlineField(str string) (fields CmdLineFields) {

	var currSegs = []byte{}
	var currQuot string
	var currSlash = false

	var preField = []CmdLineSeg{}

	for _, c := range []byte(str) {

		if currSlash {
			currSegs = append(currSegs, c)
		} else {

			if currQuot != "" {
				currSegs = append(currSegs, c)
				if c == currQuot[0] {
					seg := CmdLineSeg{content: string(currSegs), quoted: true}
					preField = append(preField, seg)
					currSegs = []byte{}
					currQuot = ""
				}
			} else {

				for _, q := range rawStrBounds {
					if c == q {
						currQuot = string(q)
						break
					}
				}

				if currQuot != "" {
					if len(currSegs) != 0 {
						seg := CmdLineSeg{content: string(currSegs), quoted: false}
						preField = append(preField, seg)
					}
					currSegs = []byte{c}
				} else {
					currSegs = append(currSegs, c)
				}
			}

		}

		if c == '\\' {
			currSlash = !currSlash
		} else {
			currSlash = false
		}
	}

	if len(currSegs) != 0 {
		preField = append(preField, CmdLineSeg{content: string(currSegs), quoted: false})
	}

	// result := []string{}

	for _, seg := range preField {
		var content string
		if seg.quoted {
			if len(seg.content) <= 2 {
				content = ""
			} else {
				content = seg.content[1 : len(seg.content)-1]
				for _, q := range rawStrBounds {
					content = strings.ReplaceAll(content, string([]byte{'\\', q}), string(q))
				}
			}
			fields.Append(CmdLineSeg{content: content, quoted: true})

		} else {
			content = seg.content
			for _, q := range rawStrBounds {
				content = strings.ReplaceAll(content, string([]byte{'\\', q}), string(q))
			}
			for _, s := range strings.Fields(content) {
				fields.Append(CmdLineSeg{content: s, quoted: false})
			}
		}

	}

	return
}

func LongestCommonPrefix(strs []string) (prefix string) {

	var prefixLen int = 1

	if len(strs) == 0 {
		return ""
	} else if len(strs) == 1 {
		return strs[0]
	}

	cnt := 0

	for {
		cnt = 0
		for _, s := range strs {

			if len(s) < prefixLen {
				break
			}

			if !strings.HasPrefix(s, strs[0][:prefixLen]) {
				break
			}

			cnt++
		}
		if cnt != len(strs) {
			break
		}
		prefixLen++
	}

	prefixLen--

	if prefixLen == 0 {
		return ""
	} else {
		return strs[0][:prefixLen]
	}
}

func RangeNumParse(_raw string, do func(uint32)) (err error) {

	raw := strings.ToLower(_raw)
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, ";")

	reCheck := regexp.MustCompile(`[0-9]+(?:-[0-9]+)?(?:;[0-9]+(?:-[0-9]+)?)*`)

	if !reCheck.MatchString(raw) {
		return fmt.Errorf("invalid number ranges: %s", _raw)
	}

	segs := strings.Split(raw, ";")

	for _, seg := range segs {

		seg = strings.TrimSpace(seg)
		nums := strings.Split(seg, "-")

		switch len(nums) {
		case 1:
			a, e := strconv.Atoi(nums[0])
			if e == nil {
				do(uint32(a))
			}
		case 2:
			a, e1 := strconv.Atoi(nums[0])
			b, e2 := strconv.Atoi(nums[1])
			if e1 == nil || e2 == nil {

				if a > b {
					a, b = b, a
				}

				for i := a; i <= b; i++ {
					do(uint32(i))
				}

			}
		}

	}

	return
}
