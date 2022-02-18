package completer

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"
	"text/tabwriter"
)

type paramType int

const (
	paramTypeSelection paramType = iota
	paramTypePlain
)

func (t paramType) String() string {
	switch t {
	case paramTypeSelection:
		return "SELECTION"
	case paramTypePlain:
		return "PLAIN"
	default:
		return "???"
	}
}

func (t *paramType) UnmarshalJSON(data []byte) (err error) {

	str := strings.ToUpper(strings.TrimSpace(string(data)))

	switch str {
	case "\"SELECTION\"":
		*t = paramTypeSelection
	case "\"PLAIN\"":
		*t = paramTypePlain
	default:
		return fmt.Errorf("invalid param type: |%s|", str)
	}

	return
}

type paramRange interface{}

type paramNameDesc struct {
	Name string `json:"name"`
	desc string
}

func (n *paramNameDesc) UnmarshalJSON(data []byte) (err error) {

	if len(data) < 2 {
		return fmt.Errorf("invalid name")
	}

	if len(data) == 2 {
		n.Name = ""
	}

	str := string(data)[1 : len(data)-1]

	str = strings.TrimSpace(str)

	splited := strings.SplitN(str, ":", 2)

	n.Name = strings.TrimSpace(splited[0])
	if len(splited) > 1 {
		n.desc = strings.TrimSpace(splited[1])
	}

	return
}

type schemaParam struct {
	NameDesc  paramNameDesc `json:"name"`
	Type      paramType     `json:"type"`
	Range     paramRange    `json:"range"`
	Optional  bool          `json:"optional"`
	Condition []string      `json:"condition"`
	Unique    bool          `json:"uniq"`
}

func (p *schemaParam) getHelps() (helps []cmdHelp) {

	switch p.Type {
	case paramTypeSelection:
		sels, descs, _ := rangeDecodeSelection(p.Range)

		for i := range sels {

			var info string

			if len(descs) > i {
				info = descs[i]
				if p.NameDesc.desc != "" {
					info += "(" + p.NameDesc.desc + ")"
				}
			}

			helps = append(helps, cmdHelp{whatToInput: sels[i], info: info})

		}

	case paramTypePlain:
		helps = append(helps, cmdHelp{whatToInput: "<" + p.NameDesc.Name + ">", info: p.NameDesc.desc})
	}

	return
}

func conditionCheckEqual(p *schemaParam, context *cmdContext, toCheck, given string, notEq bool) (ok bool) {

	var nameMode bool

	if !strings.HasPrefix(toCheck, "{") && !strings.HasSuffix(toCheck, "}") {
		nameMode = true
	}

	toCheck = strings.Trim(toCheck, "{}")

	var equal bool

	if strings.HasPrefix(toCheck, "-") {
		//relative mode
		//TODO: support offset number

		lastName, lastValue, ok := context.getLast()

		if !ok {
			// fmt.Printf("no context last\n")
			return notEq
		}

		if (nameMode && (lastName == given)) || (!nameMode && (lastValue == given)) {
			equal = true
		}

		// fmt.Printf("checking equal, %s, lastName %s lastValue %s, name mode %t, ok %t\n", p.NameDesc.Name, lastName, lastValue, nameMode, equal && !notEq)

	} else {

		if nameMode {
			return toCheck == given
		}

		exists := context.lookup(toCheck)

		for _, exist := range exists {
			if exist == given {
				equal = true
				break
			}
		}

	}

	return notEq == !equal
}

func conditionCheckIn(p *schemaParam, context *cmdContext, toCheck, given string, notIn bool) (ok bool) {

	var nameMode bool

	if !strings.HasPrefix(toCheck, "{") && !strings.HasSuffix(toCheck, "}") {
		nameMode = true
	}

	toCheck = strings.Trim(toCheck, "{}")

	givens := strings.Fields(given)
	var inGivens bool

	if strings.HasPrefix(toCheck, "-") {
		//relative mode
		//TODO: support offset number

		lastName, lastValue, ok := context.getLast()

		if !ok {
			return notIn
		}

		for _, g := range givens {
			if (nameMode && g == lastName) || (!nameMode && g == lastValue) {
				inGivens = true
			}
		}

	} else {

		if nameMode {
			for _, g := range givens {
				if toCheck == g {
					inGivens = true
				}
			}
		} else {

			exists := context.lookup(toCheck)
			for _, exist := range exists {
				for _, g := range givens {
					if exist == g {
						inGivens = true
						break
					}
				}
				if inGivens {
					break
				}
			}
		}
	}

	return inGivens == !notIn
}

func (p *schemaParam) conditionCheck(context *cmdContext) bool {

	// fmt.Printf("condition for param %s: %s\n", p.NameDesc.Name, p.Condition)

	if len(p.Condition) == 0 {
		return true
	}

	if p.Unique && context.count(p.NameDesc.Name) > 0 {
		return false
	}

	condEqual, _ := regexp.Compile(`(\{?\S+\}?)\s*(?:(not)\s+)?eq\s+(\S+)\s*`)
	condIn, _ := regexp.Compile(`(\{?\S+\}?)\s*(?:(not)\s+)?in\s+(.*)`)

	for _, c := range p.Condition {

		var segs []string

		if c == "*" {
			//means any conditions
			continue
		}

		segs = condEqual.FindStringSubmatch(c)
		if segs != nil {
			toCheck := segs[1]
			given := segs[3]

			if !conditionCheckEqual(p, context, toCheck, given, segs[2] == "not") {
				return false
			} else {
				// fmt.Printf("cond %s eq check of param %s: toCheck %s given %s, ok\n", segs[2], p.NameDesc.Name, toCheck, given)
			}

			continue
		}

		segs = condIn.FindStringSubmatch(c)
		if segs != nil {
			toCheck := segs[1]
			given := segs[3]

			if !conditionCheckIn(p, context, toCheck, given, segs[2] == "not") {
				return false
			} else {
				// fmt.Printf("cond %s in check of param %s: toCheck %s given %s, ok\n", segs[2], p.NameDesc.Name, toCheck, given)
			}

			continue
		}

		// fmt.Printf("condition format invalid: %s\n", c)
		return false
	}

	return true
}

func rangeDecodeSelection(r paramRange) (names []string, descs []string, err error) {
	var sels []string

	rangeType := reflect.TypeOf(r)
	if rangeType == nil {
		return
	}

	switch rangeType.Kind() {
	case reflect.Slice:
		for _, v := range r.([]interface{}) {
			if reflect.TypeOf(v).Kind() == reflect.String {
				sels = append(sels, v.(string))
			}
		}
	case reflect.String:
		sels = []string{r.(string)}
	default:
		return names, descs, fmt.Errorf("range type invalid")
	}

	for _, s := range sels {

		s = strings.TrimSpace(s)

		fields := strings.SplitN(s, ":", 2)

		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}

		switch len(fields) {
		case 1:
			if fields[0] == "" {
				names = append(names, "")
				descs = append(descs, "")
			} else {
				if s[0] == ':' {
					names = append(names, "")
					descs = append(descs, fields[0])
				} else {
					names = append(names, fields[0])
					descs = append(descs, "")
				}
			}
		case 2:

			names = append(names, fields[0])
			descs = append(descs, fields[1])

		default:
			return names, descs, fmt.Errorf("exception")
		}
	}

	return
}

func (param *schemaParam) checkValue(value string) bool {

	switch param.Type {

	case paramTypePlain:
		//Plain type can handle any kind of value
		return true
	case paramTypeSelection:
		sels, _, _ := rangeDecodeSelection(param.Range)
		if strings.Contains(" "+strings.ToLower(strings.Join(sels, " "))+" ", strings.ToLower(" "+strings.TrimSpace(value)+" ")) {
			return true
		}
	}

	return false
}

func (param *schemaParam) getCompletions(src string) (completions []string) {

	// fmt.Printf("getting completions of param: %s\n", param.NameDesc.Name)

	if param.Type == paramTypePlain {

		return

	} else if param.Type == paramTypeSelection {

		sels, _, _ := rangeDecodeSelection(param.Range)

		if len(sels) == 0 {
			return
		}

		sels = stringsUniq(sels)

		for _, s := range sels {
			if strings.HasPrefix(strings.ToLower(s), strings.ToLower(src)) {
				completions = append(completions, s)
			}
		}

		for i := 0; i < len(completions); i++ {
			// fmt.Printf("comp %d is |%s| was |%s| src is |%s|\n", i, completions[i][len(src):]+" ", completions[i], src)
			if len(completions[i]) == len(src) {
				completions[i] = " "
			} else {
				completions[i] = completions[i][len(src):] + " "
			}
		}
	}

	return
}

type schemaCommand struct {
	Name         string        `json:"name"`
	Prefix       string        `json:"prefix"`
	Params       []schemaParam `json:"param"`
	Comment      string        `json:"comment"`
	staticParams []*schemaParam
	dynamParams  []*schemaParam
}

func (c *schemaCommand) prefixComplete(inputs *[]string, completeNext bool) (completeStr string, fulls string, prefixMatch bool) {

	inputsLen := len(*inputs)
	prefixSegs := strings.Fields(c.Prefix)

	if len(prefixSegs) == 0 {
		return
	}

	for i := 0; i < len(prefixSegs) && i < inputsLen; i++ {
		if strings.Index(strings.ToLower(prefixSegs[i]), strings.ToLower((*inputs)[i])) != 0 {
			return
		}
	}

	prefixMatch = true

	if inputsLen > len(prefixSegs) {

		*inputs = (*inputs)[len(prefixSegs):]

	} else {

		if inputsLen == 0 {
			*inputs = []string{}
			completeStr = prefixSegs[0] + " "
			fulls = prefixSegs[0]
			return
		} else {

			lastInput := (*inputs)[inputsLen-1]

			*inputs = []string{}

			if completeNext {

				if inputsLen < len(prefixSegs) {
					completeStr = prefixSegs[inputsLen] + " "
					fulls = prefixSegs[inputsLen]
					return
				}

			} else {
				completeStr = (prefixSegs[inputsLen-1] + " ")[len(lastInput):]
				fulls = prefixSegs[inputsLen-1]
				return
			}
		}

	}

	return
}

type logicPath struct {
	command        *schemaCommand
	param          *schemaParam
	context        *cmdContext
	nexts          list.List
	staticParamPos int
	invalid        *bool
	inputVal       string
}

func (path *logicPath) addNext(param *schemaParam, staticParamPos int) {
	path.nexts.PushBack(newLogicalPath(path.command, param, staticParamPos, path.context.clone()))
}

func (path *logicPath) step(values []string, next bool) (thisPath *logicPath) {

	thisPath = path

	if path.param == nil {

		if len(path.command.staticParams) != 0 {
			path.addNext(path.command.staticParams[0], 0)
		}
		for _, p := range path.command.dynamParams {
			if p.conditionCheck(path.context) {
				path.addNext(p, 0)
			}
		}

		elem := path.nexts.Front()
		for elem != nil {
			elem.Value.(*logicPath).step(values, next)
			elem = elem.Next()
		}

		return
	}

	if len(values) == 0 {
		return
	}

	path.inputVal = values[0]
	path.context.append(path.param.NameDesc.Name, values[0])

	if !path.param.checkValue(values[0]) {
		if len(values) != 1 || next {
			*path.invalid = true
		}
		return
	} else {
		if len(values) == 1 && !next {
			return
		}
	}

	if path.staticParamPos+1 < len(path.command.staticParams) {
		for _, p := range path.command.staticParams {
			if p == path.param {
				path.addNext(path.command.staticParams[path.staticParamPos+1], path.staticParamPos+1)
			}
		}
	}

	for _, p := range path.command.dynamParams {
		if p.conditionCheck(path.context) {
			path.addNext(p, path.staticParamPos)
		}
	}

	if len(values) > 1 {

		if path.nexts.Len() == 0 {
			*path.invalid = true
			return
		}

		elem := path.nexts.Front()
		for elem != nil {
			elem.Value.(*logicPath).step(values[1:], next)
			elem = elem.Next()
		}
	}

	return
}

func (path *logicPath) getComplete() (completions []string) {

	if *path.invalid {
		return
	}

	if path.nexts.Len() == 0 {

		return append(completions, path.param.getCompletions(path.inputVal)...)

	} else {
		elem := path.nexts.Front()
		for elem != nil {
			child := elem.Value.(*logicPath)
			completions = append(completions, child.getComplete()...)
			elem = elem.Next()
		}
	}

	return
}

type cmdHelp struct {
	whatToInput string
	info        string
}

func (path *logicPath) getHelps(next bool) (helps []cmdHelp) {

	if *path.invalid {
		return
	}

	if path.nexts.Len() == 0 {

		if !next || next && path.inputVal == "" {
			return path.param.getHelps()
		}

	} else {
		elem := path.nexts.Front()
		for elem != nil {
			child := elem.Value.(*logicPath)
			helps = append(helps, child.getHelps(next)...)
			elem = elem.Next()
		}
	}

	return
}

func newLogicalPath(command *schemaCommand, param *schemaParam, staticParamPos int, context *cmdContext) (path *logicPath) {

	path = new(logicPath)
	pathInvalid := false

	path.invalid = &pathInvalid

	path.command = command
	path.param = param
	path.staticParamPos = staticParamPos

	if context == nil {
		path.context = new(cmdContext)
		path.context.init()
	} else {
		path.context = context
	}

	return
}

func (c *schemaCommand) paramsComplete(inputs *[]string, completeNext bool) (ret []string) {

	if len(c.Params) == 0 {
		return
	}

	rootPath := newLogicalPath(c, nil, 0, nil)

	rootPath.step(*inputs, completeNext)

	comps := rootPath.getComplete()

	for _, c := range comps {
		if c == "" {
			if completeNext {
				return
			} else {
				return []string{" "}
			}
		} else if c == " " && completeNext {
			return
		}
	}

	return comps
}

func (c *schemaCommand) paramsHelp(inputs *[]string, completeNext bool) (helps []cmdHelp) {

	if len(c.Params) == 0 {
		return
	}

	return newLogicalPath(c, nil, 0, nil).step(*inputs, completeNext).getHelps(completeNext)
}

func (c *schemaCommand) complete(inputs []string, next bool) (completions []string) {

	prefixComp, _, match := c.prefixComplete(&inputs, next)

	if !match {
		return
	}

	if prefixComp != "" {
		return []string{prefixComp}
	}

	return c.paramsComplete(&inputs, next)
}

func (c *schemaCommand) help(inputs []string, next bool) (helps []cmdHelp) {

	_, prefixHelp, match := c.prefixComplete(&inputs, next)

	if match {
		if prefixHelp == "" {
			helps = c.paramsHelp(&inputs, next)
		} else {
			helps = append(helps, cmdHelp{whatToInput: prefixHelp, info: strings.Title(c.Name)})
		}
	}

	return
}

type schemaTop struct {
	Commands []schemaCommand `json:"commands"`
}

type Completer struct {
	source []byte
	schema schemaTop
}

func (s *Completer) Setup(filePath string) (err error) {
	s.source, err = ioutil.ReadFile(filePath)
	if err != nil {
		return
	}

	re := regexp.MustCompile(`(?m)^\s*//.*$`)

	err = json.Unmarshal(re.ReplaceAll(s.source, []byte{}), &s.schema)

	//internal registered commands
	s.schema.Commands = append(s.schema.Commands, cmdRegList...)

	if err != nil {
		return fmt.Errorf("completer config file load error, %s", err.Error())
	}

	for i := range s.schema.Commands {
		c := &s.schema.Commands[i]
		for j := range c.Params {
			pp := &c.Params[j]

			if len(pp.Condition) == 0 {
				c.staticParams = append(c.staticParams, pp)
			} else {
				c.dynamParams = append(c.dynamParams, pp)
			}
		}
	}

	return
}

func (s *Completer) GetCompletes(input string) (completions []string) {

	next := strings.HasSuffix(input, " ") //get "completions" of next param if input is end with space(s), otherwise, get "completions" of "this" param
	segs := CmdlineField(input).Strings() //split input into segments. TODO: handle unclosed quots/brackets/...

	for _, command := range s.schema.Commands {
		//combine "completions" from each commands
		completions = append(completions, command.complete(segs, next)...)
	}

	completions = stringsUniq(completions) //remove duplicated "completions"

	if commonPrefix := LongestCommonPrefix(completions); len(commonPrefix) != 0 {
		return []string{commonPrefix} //found "loggest common prefix" and return it
	}

	for _, comp := range completions {
		if comp == " " {
			return []string{comp} //found a "completion" which is a single space, return it
		}
	}

	return
}

func (s *Completer) GetHelps(input string) (helpStr string) {

	next := strings.HasSuffix(input, " ")
	segs := CmdlineField(input).Strings() //split input into segments. TODO: handle unclosed quots/brackets/...

	var helps []cmdHelp

	for _, cmd := range s.schema.Commands {
		helps = append(helps, cmd.help(segs, next)...)
	}

	mergeMap := make(map[string][]string)
	var keyOrdered []string

	for _, h := range helps {
		list, exist := mergeMap[h.whatToInput]
		mergeMap[h.whatToInput] = append(list, h.info)
		if !exist {
			keyOrdered = append(keyOrdered, h.whatToInput)
		}
	}

	var buf bytes.Buffer
	tw := tabwriter.NewWriter(&buf, 16, 8, 4, ' ', 0)

	for _, k := range keyOrdered {
		if v, ok := mergeMap[k]; ok {
			v = stringsUniq(v)
			fmt.Fprintf(tw, "%s\t%s\n", k, strings.Join(v, " / "))
		}
	}

	tw.Flush()

	return buf.String()
}

//dynamic command insert
var cmdRegList []schemaCommand

func RegisterCmd(raw []byte) {
	var cmd schemaCommand
	re := regexp.MustCompile(`(?m)^\s*//.*$`)
	if err := json.Unmarshal(re.ReplaceAll(raw, []byte{}), &cmd); err != nil {
		return
	}
	cmdRegList = append(cmdRegList, cmd)
}
