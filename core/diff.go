// diff.go — The diffing engine. Compares old and new virtual DOM trees
// and produces a minimal patch set to apply to the real DOM.
package core

// ---------------------------------------------------------
// PatchType — what kind of DOM operation is needed
// ---------------------------------------------------------

type PatchType int

const (
	PatchCreate  PatchType = iota // create a new DOM node
	PatchRemove                   // remove a DOM node
	PatchReplace                  // replace a node with a different type
	PatchUpdate                   // update props/attrs on existing node
	PatchText                     // update text content
	PatchMove                     // reorder a node (keyed lists)
	PatchNone                     // no change needed
)

// Patch describes a single change to apply to the real DOM
type Patch struct {
	Type    PatchType
	OldNode *Node
	NewNode *Node
	Index   int    // child index
	Key     string // for keyed patches
}

// ---------------------------------------------------------
// Diff — compares two virtual DOM trees
// ---------------------------------------------------------

// Diff compares old and new virtual DOM nodes and returns
// the list of patches needed to bring the real DOM up to date.
// This is the core of RyxoGo's rendering efficiency.
func Diff(old, new *Node) []Patch {
	var patches []Patch
	diff(old, new, &patches)
	return patches
}

func diff(old, new *Node, patches *[]Patch) {
	// Case 1: Both nil — nothing to do
	if old == nil && new == nil {
		return
	}

	// Case 2: New node added — create it
	if old == nil {
		*patches = append(*patches, Patch{
			Type:    PatchCreate,
			NewNode: new,
		})
		return
	}

	// Case 3: Node removed
	if new == nil {
		*patches = append(*patches, Patch{
			Type:    PatchRemove,
			OldNode: old,
		})
		return
	}

	// Case 4: Node type changed — full replace
	if old.Type != new.Type {
		*patches = append(*patches, Patch{
			Type:    PatchReplace,
			OldNode: old,
			NewNode: new,
		})
		return
	}

	// Case 5: Text node changed
	if old.Type == TextNode {
		if old.Text != new.Text {
			*patches = append(*patches, Patch{
				Type:    PatchText,
				OldNode: old,
				NewNode: new,
			})
		}
		return
	}

	// Case 6: Different tag — replace
	if old.Tag != new.Tag {
		*patches = append(*patches, Patch{
			Type:    PatchReplace,
			OldNode: old,
			NewNode: new,
		})
		return
	}

	// Case 7: Same tag — check if props changed
	if propsChanged(old.Props, new.Props) {
		*patches = append(*patches, Patch{
			Type:    PatchUpdate,
			OldNode: old,
			NewNode: new,
		})
	}

	// Case 8: Diff children
	diffChildren(old.Children, new.Children, patches)
}

// diffChildren handles child node comparison, including keyed lists
func diffChildren(oldChildren, newChildren []*Node, patches *[]Patch) {
	// Check if any children have keys (for efficient list diffing)
	hasKeys := false
	for _, c := range newChildren {
		if c != nil && c.Key != "" {
			hasKeys = true
			break
		}
	}

	if hasKeys {
		diffKeyedChildren(oldChildren, newChildren, patches)
		return
	}

	// Simple index-based diff for non-keyed children
	maxLen := len(oldChildren)
	if len(newChildren) > maxLen {
		maxLen = len(newChildren)
	}

	for i := 0; i < maxLen; i++ {
		var oldChild, newChild *Node
		if i < len(oldChildren) {
			oldChild = oldChildren[i]
		}
		if i < len(newChildren) {
			newChild = newChildren[i]
		}
		diff(oldChild, newChild, patches)
	}
}

// diffKeyedChildren uses keys for O(n) list diffing.
// This makes reordering lists fast — React requires keys for the same reason.
func diffKeyedChildren(oldChildren, newChildren []*Node, patches *[]Patch) {
	// Build map of old children by key
	oldByKey := make(map[string]*Node)
	for _, c := range oldChildren {
		if c != nil && c.Key != "" {
			oldByKey[c.Key] = c
		}
	}

	// Match new children to old by key
	for i, newChild := range newChildren {
		if newChild == nil {
			continue
		}
		if newChild.Key == "" {
			// No key — fall back to index diff
			var oldChild *Node
			if i < len(oldChildren) {
				oldChild = oldChildren[i]
			}
			diff(oldChild, newChild, patches)
			continue
		}

		oldChild, exists := oldByKey[newChild.Key]
		if !exists {
			// New keyed node — create it
			*patches = append(*patches, Patch{
				Type:    PatchCreate,
				NewNode: newChild,
				Key:     newChild.Key,
			})
		} else {
			// Existing key — diff it
			diff(oldChild, newChild, patches)
			delete(oldByKey, newChild.Key)
		}
	}

	// Remove old keyed nodes that no longer exist
	for key, oldChild := range oldByKey {
		*patches = append(*patches, Patch{
			Type:    PatchRemove,
			OldNode: oldChild,
			Key:     key,
		})
	}
}

// propsChanged checks if two Props are different
func propsChanged(a, b Props) bool {
	if a.ID != b.ID { return true }
	if a.Class != b.Class { return true }
	if a.Value != b.Value { return true }
	if a.Placeholder != b.Placeholder { return true }
	if a.Disabled != b.Disabled { return true }
	if a.Checked != b.Checked { return true }
	if a.Type != b.Type { return true }
	if a.Src != b.Src { return true }
	if a.Alt != b.Alt { return true }
	if a.Href != b.Href { return true }

	// Check style map
	if len(a.Style) != len(b.Style) { return true }
	for k, v := range a.Style {
		if b.Style[k] != v { return true }
	}

	// Check attr map
	if len(a.Attrs) != len(b.Attrs) { return true }
	for k, v := range a.Attrs {
		if b.Attrs[k] != v { return true }
	}

	// Note: we can't compare func values in Go directly,
	// so event handler changes always trigger a prop update.
	// The renderer handles this by re-attaching handlers.
	return false
}
