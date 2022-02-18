package history

import (
	"strings"
)

type HRing struct {
	capacity    int
	histories   *[]string
	dupstr      string
	cnt         int
	first_index int
	last_index  int
	curr_pos    int //num from last
}

func NewHRing(capacity int) (ring *HRing) {
	ring = new(HRing)
	slice := make([]string, 0, capacity)
	ring.histories = &slice
	ring.capacity = capacity
	return
}

func (h *HRing) Cnt() (cnt int) {
	return h.cnt
}

func (h *HRing) PosBack() (ok bool) {

	if h.curr_pos+1 >= h.cnt {
		//hold on first
		return true
	}

	h.curr_pos++

	return true
}

func (h *HRing) PosForward() (ok bool) {

	h.curr_pos--

	if h.curr_pos < 0 {
		h.curr_pos = -1
		return false
	}

	return true
}

func (h *HRing) Read() string {

	if h.curr_pos < 0 {
		return ""
	} else if h.curr_pos < (h.last_index + 1) {
		return (*h.histories)[h.last_index-h.curr_pos]
	} else {
		return (*h.histories)[h.capacity-(h.curr_pos-h.last_index)]
	}
}

func (h *HRing) Append(content string) {

	tempComStr := strings.ReplaceAll(strings.ReplaceAll(content, " ", ""), "\n", "")

	if h.dupstr == tempComStr {
		h.curr_pos = -1
		return
	}

	if h.cnt < h.capacity {
		appended := append((*h.histories), content)
		h.histories = &appended
		if h.cnt != 0 {
			h.last_index++
		}
		h.cnt++
	} else {

		//move first index
		if h.first_index == h.capacity-1 {
			h.first_index = 0
		} else {
			h.first_index++
		}

		//move_last_index
		if h.last_index == h.capacity-1 {
			h.last_index = 0
		} else {
			h.last_index++
		}

		(*h.histories)[h.last_index] = content
	}
	h.dupstr = tempComStr
	h.curr_pos = -1
}
