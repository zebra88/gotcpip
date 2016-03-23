package rbtree

import (
	"reflect"
	"slab"
	"unsafe"
)

type Item interface{}

const BLACK = 0
const RED = 1

var rbPool *slab.SyncPool

type Node struct {
	keyValue   Item /* generic key */
	parent     *Node
	leftChild  *Node
	rightChild *Node
	color      uint8
	addr       []byte
}

type CompareFunc func(keyA, keyB Item) int

type RBtree struct {
	root *Node /* root of the tree */
	//	minNode *Node
	//	maxNode *Node
	//	size    uint
	/* this function directly provides the keys as parameters not the nodes. */
	Compare CompareFunc
}

func Byte2Pointer(b []byte) unsafe.Pointer {
	return unsafe.Pointer(
		(*reflect.SliceHeader)(unsafe.Pointer(&b)).Data,
	)
}

const NODESIZE = int(unsafe.Sizeof(Node{}))

/* generic leaf node */
var LEAF = Node{
	keyValue:   nil, /* key */
	parent:     nil,
	leftChild:  nil,
	rightChild: nil,   /* parent, left,right */
	color:      BLACK, /* color */
}

func InitTree(compareFunction CompareFunc) *RBtree {

	tree := &RBtree{
		root:    &LEAF,
		Compare: compareFunction,
	}
	return tree
}
func IS_LEAF(x *Node) bool {
	return x == &LEAF
}

func IS_NOT_LEAF(x *Node) bool {
	return (x != &LEAF)
}

//const INIT_LEAF = (&LEAF)
var INIT_LEAF *Node = (&LEAF)

func AM_I_LEFT_CHILD(x *Node) bool {
	return (x == x.parent.leftChild)
}

func AM_I_RIGHT_CHILD(x *Node) bool {
	return (x == x.parent.rightChild)
}

func PARENT(x *Node) *Node {
	return (x.parent)
}

func GRANPA(x *Node) *Node {
	return (x.parent.parent)
}

func TreeFirstNode(node *Node) *Node {
	for IS_NOT_LEAF(node.leftChild) {
		node = node.leftChild
	}

	return node
}

func TreeLastNode(node *Node) *Node {
	for IS_NOT_LEAF(node.rightChild) {
		node = node.rightChild
	}
	return node
}

func TreeNext(node *Node) *Node {

	if node == nil {
		return nil
	}
	if IS_NOT_LEAF(node.rightChild) {
		node = node.rightChild
		for IS_NOT_LEAF(node.leftChild) {
			node = node.leftChild
		}
	} else {
		if IS_NOT_LEAF(node.parent) && AM_I_LEFT_CHILD(node) {
			node = node.parent
		} else {
			for IS_NOT_LEAF(node.parent) && AM_I_RIGHT_CHILD(node) {
				node = node.parent
			}
			node = node.parent
		}
	}

	return node
}

func TreePrev(node *Node) *Node {

	if IS_NOT_LEAF(node.leftChild) {
		node = node.leftChild
		for IS_NOT_LEAF(node.rightChild) {
			node = node.rightChild
		}
	} else {
		if IS_NOT_LEAF(node.parent) && AM_I_RIGHT_CHILD(node) {
			node = node.parent
		} else {
			for IS_NOT_LEAF(node) && AM_I_LEFT_CHILD(node) {
				node = node.parent
			}
			node = node.parent
		}
	}

	return node
}

func (tree *RBtree) FindNode(key Item) *Node {

	found := tree.root

	for IS_NOT_LEAF(found) {

		result := tree.Compare(found.keyValue, key)
		if result == 0 {
			return found
		} else if result < 0 {
			found = found.rightChild
		} else {
			found = found.leftChild
		}
	}
	return nil
}

func (tree *RBtree) Empty() bool {
	return tree.root == nil || IS_LEAF(tree.root)
}

func (tree *RBtree) FindKey(key Item) Item {

	var found *Node

	found = tree.root
	for IS_NOT_LEAF(found) {
		/*
			if IS_LEAF(found) {
				break
			}
		*/
		result := tree.Compare(found.keyValue, key)
		if result == 0 {
			return found.keyValue
		} else if result < 0 {
			found = found.rightChild
		} else {
			found = found.leftChild
		}

	}
	return nil
}

func (tree *RBtree) Insert(key Item) Item {

	last_node := INIT_LEAF
	temp := tree.root
	var insert *Node
	var LocalKey Item
	result := int(0)

	if IS_NOT_LEAF(tree.root) {
		LocalKey = tree.FindKey(key)
	} else {
		LocalKey = nil
	}

	/* if node already in, bail out */
	if LocalKey != nil {
		return LocalKey
	} else {
		insert = tree.createNode(key)

		if insert == nil {
			/* to let the user know that it couldn't insert */
			return Item(&LEAF)
		}
	}

	/* search for the place to insert the new node */
	for IS_NOT_LEAF(temp) {
		last_node = temp
		result = tree.Compare(insert.keyValue, temp.keyValue)
		if result < 0 {
			temp = temp.leftChild
		} else {
			temp = temp.rightChild
		}
	}

	/* make the needed connections */
	insert.parent = last_node

	if IS_LEAF(last_node) {
		tree.root = insert
	} else {
		result = tree.Compare(insert.keyValue, last_node.keyValue)
		if result < 0 {
			last_node.leftChild = insert
		} else {
			last_node.rightChild = insert
		}
	}

	/* fix colour issues */
	tree.fixInsertCollisions(insert)
	return nil
}

func (tree *RBtree) Delete(key Item) Item {

	if key == nil {
		return nil
	}
	delete := tree.FindNode(key)

	/* this key isn't in the tree, bail out */
	if delete == nil {
		return nil
	}
	var temp *Node
	lkey := delete.keyValue
	nodeColor := tree.deleteCheckSwitch(delete, &temp)

	/* deleted node is black, this will mess up the black path property */
	if nodeColor == BLACK {
		tree.fixDeleteCollisions(temp)
	}

	RBNodeFree(delete)
	return lkey
}

func (tree *RBtree) deleteCheckSwitch(delete *Node, temp **Node) uint8 {
	ltemp := delete
	nodeColor := delete.color
	if IS_LEAF(delete.leftChild) {
		*temp = ltemp.rightChild
		tree.switchNodes(ltemp, ltemp.rightChild)
	} else {
		if IS_LEAF(delete.rightChild) {
			_ltemp := delete
			*temp = _ltemp.leftChild
			tree.switchNodes(_ltemp, _ltemp.leftChild)
		} else {
			nodeColor = tree.deleteNode(delete, temp)
		}
	}
	return nodeColor
}

func (tree *RBtree) deleteNode(d *Node, temp **Node) uint8 {

	ltemp := d
	min := TreeFirstNode(d.rightChild)
	nodeColor := min.color

	*temp = min.rightChild
	if min.parent == ltemp && IS_NOT_LEAF(*temp) {
		(*temp).parent = min
	} else {
		tree.switchNodes(min, min.rightChild)
		min.rightChild = ltemp.rightChild
		if IS_NOT_LEAF(min.rightChild) {
			min.rightChild.parent = min
		}
	}

	tree.switchNodes(ltemp, min)
	min.leftChild = ltemp.leftChild

	if IS_NOT_LEAF(min.leftChild) {
		min.leftChild.parent = min
	}
	min.color = ltemp.color
	return nodeColor
}

func (tree *RBtree) switchNodes(nodeA *Node, nodeB *Node) {

	if IS_LEAF(nodeA.parent) {
		tree.root = nodeB
	} else {
		if IS_NOT_LEAF(nodeA) {
			if AM_I_LEFT_CHILD(nodeA) {
				nodeA.parent.leftChild = nodeB
			} else {
				nodeA.parent.rightChild = nodeB
			}
		}
	}
	if IS_NOT_LEAF(nodeB) {
		nodeB.parent = nodeA.parent
	}

}

/*
 * This function fixes the possible collisions in the tree.
 * Eg. if a node is red his children must be black !
 */
func (tree *RBtree) fixInsertCollisions(node *Node) {
	var temp *Node

	for node.parent.color == RED && IS_NOT_LEAF(GRANPA(node)) {
		if AM_I_RIGHT_CHILD(node.parent) {
			temp := GRANPA(node).leftChild
			if temp.color == RED {
				node.parent.color = BLACK
				temp.color = BLACK
				GRANPA(node).color = RED
				node = GRANPA(node)
			} else if temp.color == BLACK {
				if node == node.parent.leftChild {
					node = node.parent
					rotateToRight(tree, node)
				}

				node.parent.color = BLACK
				GRANPA(node).color = RED
				rotateToLeft(tree, GRANPA(node))
			}
		} else if AM_I_LEFT_CHILD(node.parent) {
			temp = GRANPA(node).rightChild
			if temp.color == RED {
				node.parent.color = BLACK
				temp.color = BLACK
				GRANPA(node).color = RED
				node = GRANPA(node)
			} else if temp.color == BLACK {
				if AM_I_RIGHT_CHILD(node) {
					node = node.parent
					rotateToLeft(tree, node)
				}

				node.parent.color = BLACK
				GRANPA(node).color = RED
				rotateToRight(tree, GRANPA(node))
			}
		}
	}
	/* make sure that the root of the tree stays black */
	tree.root.color = BLACK
}

func (tree *RBtree) fixDeleteCollisions(node *Node) {
	var temp *Node

	for node != tree.root && node.color == BLACK && IS_NOT_LEAF(node) {
		if AM_I_LEFT_CHILD(node) {

			temp = node.parent.rightChild
			if temp.color == RED {
				temp.color = BLACK
				node.parent.color = RED
				rotateToLeft(tree, node.parent)
				temp = node.parent.rightChild
			}

			if temp.leftChild.color == BLACK && temp.rightChild.color == BLACK {
				temp.color = RED
				node = node.parent
			} else {
				if temp.rightChild.color == BLACK {
					temp.leftChild.color = BLACK
					temp.color = RED
					rotateToRight(tree, temp)
					temp = temp.parent.rightChild
				}

				temp.color = node.parent.color
				node.parent.color = BLACK
				temp.rightChild.color = BLACK
				rotateToLeft(tree, node.parent)
				node = tree.root
			}
		} else {
			temp = node.parent.leftChild
			if temp.color == RED {
				temp.color = BLACK
				node.parent.color = RED
				rotateToRight(tree, node.parent)
				temp = node.parent.leftChild
			}

			if temp.rightChild.color == BLACK && temp.leftChild.color == BLACK {
				temp.color = RED
				node = node.parent
			} else {
				if temp.leftChild.color == BLACK {
					temp.rightChild.color = BLACK
					temp.color = RED
					rotateToLeft(tree, temp)
					temp = temp.parent.leftChild
				}

				temp.color = node.parent.color
				node.parent.color = BLACK
				temp.leftChild.color = BLACK
				rotateToRight(tree, node.parent)
				node = tree.root
			}
		}
	}
	node.color = BLACK
}

func rotateToLeft(tree *RBtree, node *Node) {

	temp := node.rightChild

	if temp == &LEAF {
		return
	}
	node.rightChild = temp.leftChild

	if IS_NOT_LEAF(temp.leftChild) {
		temp.leftChild.parent = node
	}
	temp.parent = node.parent

	if IS_LEAF(node.parent) {
		tree.root = temp
	} else {
		if node == node.parent.leftChild {
			node.parent.leftChild = temp
		} else {
			node.parent.rightChild = temp
		}
	}
	temp.leftChild = node
	node.parent = temp
}

func rotateToRight(tree *RBtree, node *Node) {

	temp := node.leftChild
	node.leftChild = temp.rightChild

	if temp == &LEAF {
		return
	}

	if IS_NOT_LEAF(temp.rightChild) {
		temp.rightChild.parent = node
	}

	temp.parent = node.parent

	if IS_LEAF(node.parent) {
		tree.root = temp
	} else {
		if node == node.parent.rightChild {
			node.parent.rightChild = temp
		} else {
			node.parent.leftChild = temp
		}
	}
	temp.rightChild = node
	node.parent = temp
	return
}

func (tree *RBtree) createNode(key Item) *Node {

	node := RBNodeAlloc()
	if node == nil {
		return nil
	}
	node.keyValue = key
	node.parent = &LEAF
	node.leftChild = &LEAF
	node.rightChild = &LEAF
	/* by default every new node is red */
	node.color = RED
	return node
}

func RBNodeAlloc() *Node {
	temp := rbPool.Alloc(NODESIZE)
	node := (*Node)(Byte2Pointer(temp))
	node.addr = temp
	return node
}

func RBNodeFree(node *Node) {
	rbPool.Free(node.addr)
}

func InitRBtreeMemPool() {
	rbPool = slab.NewSyncPool(
		60,          // The smallest chunk size is 64B.
		60*1024*200, // The largest chunk size is 64KB.
		2,           // Power of 2 growth in chunk size.
	)

}
