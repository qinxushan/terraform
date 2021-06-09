package globalref

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang"
)

// ReferencesFromResource returns all of the direct references from the
// definition of the resource instance at the given address. It doesn't
// include any indirect references.
//
// Resource configurations can only refer to other objects within the same
// module, so callers should assume that the returned references are all
// relative to the same module instance that the given address belongs to.
func (a *Analyzer) ReferencesFromResourceInstance(addr addrs.AbsResourceInstance) []Reference {
	// Using MetaReferences for this is kinda overkill, since
	// lang.ReferencesInBlock would be sufficient really, but
	// this ensures we keep consistent and aside from some
	// extra overhead this call boils down to a call to
	// lang.ReferencesInBlock anyway.
	fakeRef := Reference{
		ContainerAddr: addr.Module,
		LocalRef: &addrs.Reference{
			Subject: addr.Resource,
		},
	}
	return a.MetaReferences(fakeRef)
}

// ReferencesFromResourceRepetition returns the references from the given
// resource's for_each or count expression, or an empty set if the resource
// doesn't use repetition.
//
// This is a special-case sort of helper for use in situations where an
// expression might refer to count.index, each.key, or each.value, and thus
// we say that it depends indirectly on the repetition expression.
func (a *Analyzer) ReferencesFromResourceRepetition(addr addrs.AbsResource) []Reference {
	modCfg := a.ModuleConfig(addr.Module)
	if modCfg == nil {
		return nil
	}
	rc := modCfg.ResourceByAddr(addr.Resource)
	if rc == nil {
		return nil
	}

	// We're assuming here that resources can either have count or for_each,
	// but never both, because that's a requirement enforced by the language
	// decoder. But we'll assert it just to make sure we catch it if that
	// changes for some reason.
	if rc.ForEach != nil && rc.Count != nil {
		panic(fmt.Sprintf("%s has both for_each and count", addr))
	}

	switch {
	case rc.ForEach != nil:
		refs, _ := lang.ReferencesInExpr(rc.ForEach)
		return absoluteRefs(addr.Module, refs)
	case rc.Count != nil:
		refs, _ := lang.ReferencesInExpr(rc.Count)
		return absoluteRefs(addr.Module, refs)
	default:
		return nil
	}
}
