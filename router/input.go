package router

import "fmt"

type Input struct {
	subMatches []string
	raw        string
	unitName   string
}

func (c *Input) GetSegment(index int) (seg string, err error) {
	if index >= len(c.subMatches) {
		return "", fmt.Errorf("segment index out of range")
	}
	return c.subMatches[index], nil
}

func (c *Input) SegmentExist(index int) bool {
	if index >= len(c.subMatches) {
		return false
	}
	return c.subMatches[index] != ""
}

func (c *Input) GetRaw() string {
	return c.raw
}

func (c *Input) GetName() string {
	return c.unitName
}

func createInput(raw string, subMatches []string, name string) (input Input) {
	input.raw = raw
	input.subMatches = subMatches
	input.unitName = name
	return
}
