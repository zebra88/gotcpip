package rbtree

import (
	"fmt"
	"reflect"
	"testing"
	"unsafe"
)

type dns_query struct {
	len uint16
	id  uint16
}

func queryCmp(ka, kb Item) int {
	fmt.Println(reflect.TypeOf(ka))
	var a, b *dns_query
	switch v := ka.(type) {
	/*
		case string:
			fmt.Println(v)
		case int32, int64:
			fmt.Println(v)
	*/
	case *dns_query:
		a = v
	default:
		fmt.Println("unknown")
	}
	switch v := kb.(type) {
	/*
		case string:
			fmt.Println(v)
		case int32, int64:
			fmt.Println(v)
	*/
	case *dns_query:
		b = v
	default:
		fmt.Println("unknown")
	}

	fmt.Printf("======a:%p-%d,b:%p-%d\n", a, a, b, b)
	if a.id == b.id {
		return 0
	}
	if a.id < b.id {
		return -1
	} else {
		return 1
	}
}

func TestRBtree(t *testing.T) {
	InitRBtreeMemPool()
	tree := InitTree(queryCmp)
	if tree == nil {
		t.Errorf("InitTree fail")
	}
	key := &dns_query{
		len: 100,
		id:  200,
	}

	fmt.Printf("key0:%p---%v\n", key, key)
	found := tree.Insert(key)
	if found != nil {
		t.Errorf("0 Insert fail")
	}
	key = &dns_query{
		len: 200,
		id:  300,
	}
	fmt.Printf("key:%p---%v\n", &key, key)
	found = tree.Insert(key)
	if found != nil {
		t.Errorf("1 Insert fail")
	}

	test := &dns_query{id: 300}

	found = tree.FindKey(test)
	if found == nil {
		t.Errorf("FindKey fail")
	}

	found = tree.Delete(test)
	if found == nil {
		t.Errorf("Delete fail")
	}
	key = &dns_query{
		len: 200,
		id:  300,
	}
	found = tree.Insert(key)
	if found != nil {
		t.Errorf("1 Insert fail")
	}

	found = tree.FindKey(test)
	if found == nil {
		t.Errorf("2-----FindKey fail")
	}

	node := tree.FindNode(test)
	if node == nil {
		t.Errorf("2-----FindKey fail")
	}
	fmt.Println(node)
	fmt.Println(unsafe.Sizeof(node))
	fmt.Println(unsafe.Sizeof(node.addr))
}
