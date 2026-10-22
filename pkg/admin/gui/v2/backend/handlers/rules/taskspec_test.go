package rules

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/tasks" // registers the task types
)

// jsonArgs are the arguments a task reads from its structure: the JSON names
// of its fields, those of its embedded structures included.
func jsonArgs(typ reflect.Type) []string {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil
	}

	var names []string

	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Anonymous {
			names = append(names, jsonArgs(field.Type)...)

			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name != "" && name != "-" {
			names = append(names, name)
		}
	}

	return names
}

func argNames(spec *taskSpec) []string {
	names := make([]string, 0, len(spec.Args))
	for _, arg := range spec.Args {
		names = append(names, arg.Name)
	}

	return names
}

// The description of the tasks is the single source of their forms: it must
// hold every type the product knows, exactly the arguments each task reads,
// and the texts of the catalogue; TestTaskSpecsRequired checks what each task
// requires against the product.
func TestTaskSpecs(t *testing.T) {
	t.Parallel()

	t.Run("Given the types the product registers", func(t *testing.T) {
		t.Parallel()

		for typ := range model.ValidTasks {
			assert.NotNil(t, specOf(typ), "the type %s has its description", typ)
		}

		assert.Len(t, taskSpecs, len(model.ValidTasks), "no description of a type the product does not know")

		for i := range taskSpecs {
			assert.Contains(t, families, taskSpecs[i].Family, "the family of %s", taskSpecs[i].Type)
		}
	})

	t.Run("Given the tasks that read their arguments from a structure", func(t *testing.T) {
		t.Parallel()

		for i := range taskSpecs {
			spec := &taskSpecs[i]
			names := jsonArgs(reflect.TypeOf(model.ValidTasks[spec.Type]()))

			if spec.Mapped {
				assert.Empty(t, names, "%s reads its arguments from a map", spec.Type)

				continue
			}

			assert.ElementsMatch(t, names, argNames(spec), "the arguments of %s", spec.Type)
		}
	})

	t.Run("Given the catalogue", func(t *testing.T) {
		t.Parallel()

		catalog, err := locale.LoadCatalog()
		require.NoError(t, err)

		for _, family := range families {
			assert.True(t, catalog.Has(taskTypesPrefix+"family."+family), "the family %s has its title", family)
		}

		for i := range taskSpecs {
			spec := &taskSpecs[i]
			keys := []string{taskTypesPrefix + spec.Slug + ".label", taskTypesPrefix + spec.Slug + ".text"}

			for j := range spec.Args {
				arg := &spec.Args[j]
				keys = append(keys, argKey(spec, arg.Name, "label"), argKey(spec, arg.Name, "hint"))

				if arg.None != "" {
					keys = append(keys, arg.None)
				}

				if arg.Choices != nil {
					for _, c := range arg.Choices() {
						if c.Key != "" {
							keys = append(keys, c.Key)
						}
					}
				}
			}

			for _, key := range keys {
				assert.True(t, catalog.Has(key), "the catalogue has %q", key)
			}
		}
	})
}
