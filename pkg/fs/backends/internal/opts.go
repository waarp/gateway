package internal

import (
	"fmt"

	rfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
)

func SetDefaultValue(m map[string]string, key, defaultValue string) {
	if value := m[key]; value == "" {
		m[key] = defaultValue
	}
}

// WithDefaults returns the configuration of the given rclone backend: the
// given options first, then the rclone default value of every option they
// leave unset. Without it, the backend's NewFs function reads the zero value
// of every unset option instead of its default.
//
// Unlike rclone's own fs.ConfigMap, the returned configuration reads neither
// the environment nor the rclone configuration file, and only writes to opts.
func WithDefaults(backend string, opts configmap.Mapper) (*configmap.Map, error) {
	info, err := rfs.Find(backend)
	if err != nil {
		return nil, fmt.Errorf("failed to find the %q rclone backend: %w", backend, err)
	}

	config := configmap.New()
	config.AddGetter(opts, configmap.PriorityNormal)
	config.AddGetter(defaultValues(info.Options), configmap.PriorityDefault)
	config.AddSetter(opts)

	return config, nil
}

// defaultValues is a configmap.Getter returning the default value of the
// options of an rclone backend.
type defaultValues rfs.Options

func (d defaultValues) Get(key string) (string, bool) {
	opt := rfs.Options(d).Get(key)
	if opt == nil {
		return "", false
	}

	return opt.String(), true
}
