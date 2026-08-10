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

func ResolveOrPassthru[T Named](item T, itemByName map[string]T) (T, error) {
	name := item.GetName()

	if len(name) == 0 {
		return item, fmt.Errorf("%T.name = %#+v could not be resolved because it has an empty name", item, name)
	}

	if !strings.HasPrefix(name, "@") {
		return item, nil
	}

	name = strings.TrimLeft(name, "@")

	names := slices.Collect(maps.Keys(itemByName))

	resolvedItem, ok := itemByName[name]
	if !ok {
		return item, fmt.Errorf("%T.name = %#+v could not be resolved because it's not one of %v", item, name, names)
	}

	return resolvedItem, nil
}
