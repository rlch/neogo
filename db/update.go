package db

import "github.com/rlch/neogo/internal"

func SetPropValue(identifier, value any) internal.SetItem {
	return internal.SetItem{
		Identifier: identifier,
		Value:  value,
	}
}

func SetMerge(identifier, properties any) internal.SetItem {
	return internal.SetItem{
		Identifier: identifier,
		Value:  properties,
		Merge:  true,
	}
}

func SetLabels(identifier any, labels ...string) internal.SetItem {
	return internal.SetItem{
		Identifier: identifier,
		Labels: labels,
	}
}

func RemoveProp(identifier any) internal.RemoveItem {
	return internal.RemoveItem{
		Identifier: identifier,
	}
}

func RemoveLabels(identifier any, labels ...string) internal.RemoveItem {
	return internal.RemoveItem{
		Identifier: identifier,
		Labels: labels,
	}
}

func OnCreate(set ...internal.SetItem) internal.MergeOption {
	return &internal.Configurer{
		MergeOptions: func(mo *internal.MergeOptions) {
			mo.OnCreate = append(mo.OnCreate, set...)
		},
	}
}

func OnMatch(set ...internal.SetItem) internal.MergeOption {
	return &internal.Configurer{
		MergeOptions: func(mo *internal.MergeOptions) {
			mo.OnMatch = append(mo.OnMatch, set...)
		},
	}
}
