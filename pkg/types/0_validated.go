package types

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

type Validated interface {
	Validate(...string) error
}

func Validate[T Validated](items []T, extras ...string) error {
	for i, item := range items {
		err := item.Validate(extras...)
		if err != nil {
			b, _ := yaml.Marshal(item)

			return fmt.Errorf("%T[%d] failed validation because %s\n\n%s", item, i, err, string(b))
		}
	}

	return nil
}
