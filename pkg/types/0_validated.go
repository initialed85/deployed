package types

import (
	"fmt"

	"go.yaml.in/yaml/v4"
)

type Validated interface {
	Validate(...string) error
}

func ValidateOne[T Validated](item T, extras ...string) error {
	err := item.Validate(extras...)
	if err != nil {
		b, _ := yaml.Marshal(item)

		return fmt.Errorf("%s\n\n%s\n", err, string(b))
	}

	return nil
}

func ValidateMany[T Validated](items []T, extras ...string) error {
	for i, item := range items {
		err := ValidateOne(item)
		if err != nil {
			return fmt.Errorf("%T[%d] %s", item, i, err)
		}
	}

	return nil
}
