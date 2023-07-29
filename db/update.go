package db

import "github.com/rlch/neogo/internal"

func SetPropValue(entity, value any) internal.SetItem {
	return internal.SetItem{
		Entity: entity,
		Value:  value,
	}
}

func SetMerge(entity, properties any) internal.SetItem {
	return internal.SetItem{
		Entity: entity,
		Value:  properties,
		Merge:  true,
	}
}

func SetLabels(entity any, labels ...string) internal.SetItem {
	return internal.SetItem{
		Entity: entity,
		Labels: labels,
	}
}

func RemoveProp(entity any) internal.RemoveItem {
	return internal.RemoveItem{
		Entity: entity,
	}
}

func RemoveLabels(entity any, labels ...string) internal.RemoveItem {
	return internal.RemoveItem{
		Entity: entity,
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
