package stats

// Coverage represents a REST API statistics
type Coverage struct {
	UniqueHits         int                             `json:"uniqueHits"`
	ExpectedUniqueHits int                             `json:"expectedUniqueHits"`
	Percent            float64                         `json:"percent"`
	Endpoints          map[string]map[string]*Endpoint `json:"endpoints"`
}

// Endpoint represents a basic statistics structure which is used to calculate REST API coverage
type Endpoint struct {
	Params             `json:"params"`
	UniqueHits         int     `json:"uniqueHits"`
	ExpectedUniqueHits int     `json:"expectedUniqueHits"`
	Percent            float64 `json:"percent"`
	MethodCalled       bool    `json:"methodCalled"`
	Path               string  `json:"path"`
	Method             string  `json:"method"`
}

// Params represents body and query parameters
type Params struct {
	Body  *Trie `json:"body"`
	Query *Trie `json:"query"`
}

// Trie represents a coverage data
type Trie struct {
	Root               *Node `json:"root"`
	UniqueHits         int   `json:"uniqueHits"`
	ExpectedUniqueHits int   `json:"expectedUniqueHits"`
	Size               int   `json:"size"`
	Height             int   `json:"height"`
}

// NewTrie initializes Trie
func NewTrie() *Trie {
	return &Trie{
		Root: &Node{
			Children: make(map[string]*Node),
			Key:      "root",
		},
		UniqueHits: 0,
		Size:       0,
	}
}

// Add a new node to Trie
func (t *Trie) Add(key string, node *Node, leaf bool) *Node {
	if t.Size == 0 || node == nil {
		node = t.Root
	}
	depth := node.Depth + 1
	if depth > t.Height {
		t.Height = depth
	}

	node.Children[key] = &Node{
		Parent:   node,
		Children: make(map[string]*Node),
		Depth:    depth,
		Key:      key,
		IsLeaf:   leaf,
	}
	t.Size++
	if leaf {
		t.ExpectedUniqueHits++
	}

	return node.Children[key]
}

// IncreaseHits calculates hits for all nodes in given path
func (t *Trie) IncreaseHits(node *Node) {
	node.Hits++
	if node.IsLeaf && node.Hits == 1 {
		t.UniqueHits++
	}
	if node.Parent == nil {
		return
	}
	t.IncreaseHits(node.Parent)
}

// Node represents a single data unit for coverage report
type Node struct {
	Key      string           `json:"-"`
	Hits     int              `json:"hits"`
	Depth    int              `json:"-"`
	IsLeaf   bool             `json:"-"`
	Parent   *Node            `json:"-"`
	Children map[string]*Node `json:"items,omitempty"`
}

// GetChild returns child for a node
func (n *Node) GetChild(key string) *Node {
	if node, ok := n.Children[key]; ok {
		return node
	}
	return nil
}

func (n *Node) String() string {
	return n.Key
}
