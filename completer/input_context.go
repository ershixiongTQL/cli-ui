package completer

import (
	"container/list"
	"fmt"
)

type cmdContextNode struct {
	name  string
	value string
}

type cmdContext struct {
	nodes *list.List
}

func (c *cmdContext) init() {
	c.nodes = list.New()
}

func (c *cmdContext) clone() (cloned *cmdContext) {
	cloned = new(cmdContext)

	cloned.init()

	elem := c.nodes.Front()
	for elem != nil {
		n := elem.Value.(*cmdContextNode)
		cloned.append(n.name, n.value)
		elem = elem.Next()
	}
	return

}

func (c *cmdContext) UniqString() (ret string) {
	uniq := []cmdContextNode{}

	li := c.nodes.Front()
	for li != nil {

		node := li.Value.(*cmdContextNode)
		for _, u := range uniq {
			if u.name != node.name || u.value != node.value {
				uniq = append(uniq, *node)
			}
		}

		li = li.Next()
	}

	for _, u := range uniq {
		ret += fmt.Sprintf("name %s value %s\n", u.name, u.value)
	}

	return
}

func (c *cmdContext) String() (ret string) {
	li := c.nodes.Front()
	for li != nil {
		node := li.Value.(*cmdContextNode)
		ret += fmt.Sprintf("name %s value %s\n", node.name, node.value)
		li = li.Next()
	}
	return
}

//向命令上下文中添加一个节点，用于表示用户当前已输入的参数
func (c *cmdContext) append(name string, value string) {
	if c == nil {
		return
	}
	c.nodes.PushBack(&cmdContextNode{name: name, value: value})
}

func (c *cmdContext) drop() {
	if c.nodes.Len() != 0 {
		c.nodes.Remove(c.nodes.Back())
	}
}

func (c *cmdContext) walk(do func(node *cmdContextNode) (stop bool)) {

	if c == nil {
		return
	}

	elem := c.nodes.Front()
	for elem != nil {
		if do(elem.Value.(*cmdContextNode)) {
			return
		}
		elem = elem.Next()
	}
}

//在命令上下文中查询指定param已出现的次数
func (c *cmdContext) count(name string) (cnt int) {
	c.walk(func(node *cmdContextNode) (stop bool) {
		if node.name == name {
			cnt++
		}
		return
	})
	return
}

//在命令上下文中查找指定参数已存在的所有值
func (c *cmdContext) lookup(name string) (value []string) {
	c.walk(func(node *cmdContextNode) (stop bool) {
		if node.name == name {
			value = append(value, node.value)
		}
		return
	})
	return
}

//获取命令上下文中的最后一个参数
func (c *cmdContext) getLast() (name, value string, ok bool) {

	if c == nil {
		return "", "", false
	}

	elem := c.nodes.Back()
	if elem == nil {
		ok = false
		return
	}

	ctx := elem.Value.(*cmdContextNode)
	name = ctx.name
	value = ctx.value
	ok = true
	return
}
