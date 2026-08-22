package types

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type Named interface {
	GetName() string
}

func GroupByName[T Named](items []T) (map[string]T, []T, error) {
	itemByName := make(map[string]T)
	unnamedItems := make([]T, 0)

	for i, item := range items {
		name := item.GetName()

		if len(name) > 0 {
			_, existing := itemByName[name]
			if existing {
				return nil, nil, fmt.Errorf("%T[%d] has name clash on %#+v with existing %T", item, i, name, item)
			}
		} else {
			unnamedItems = append(unnamedItems, item)
		}

		itemByName[name] = item
	}

	return itemByName, unnamedItems, nil
}

func Resolve[T Named](name string, itemByName map[string]T) (T, error) {
	zeroT := *new(T)

	if len(name) == 0 {
		return zeroT, fmt.Errorf("%#+v could not be resolved because it is empty", name)
	}

	// TODO(initialed85): remove this support for legacy naming pattern
	if strings.HasPrefix(name, "@") {
		name = strings.TrimLeft(name, "@")
	}

	names := slices.Collect(maps.Keys(itemByName))

	resolvedItem, ok := itemByName[name]
	if !ok {
		return zeroT, fmt.Errorf("%#+v could not be resolved because it's not one of %v", name, names)
	}

	return resolvedItem, nil
}

func ResolveOrPassthru[T Named](item T, itemByName map[string]T) (T, error) {
	name := item.GetName()

	resolvedItem, err := Resolve(name, itemByName)
	if err != nil {
		return item, nil
	}

	return resolvedItem, nil
}
