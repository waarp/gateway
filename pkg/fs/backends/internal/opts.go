package internal

import (
	"fmt"

	rfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
)

func MakeOptions(backend string, userOpts, overrideOpts configmap.Simple) configmap.Mapper {
	fsInfo, err := rfs.Find(backend)
	if err != nil {
		panic(fmt.Sprintf("backend %q does not exist", backend))
	}

	config := configmap.New()

	// Add default options
	defaults := optionsGetter(fsInfo.Options)
	config.AddGetter(defaults, configmap.PriorityDefault)

	// Add user options
	config.AddGetter(userOpts, configmap.PriorityConfig)

	// Add override opts
	config.AddGetter(overrideOpts, configmap.PriorityNormal)
	config.AddSetter(overrideOpts)

	return config
}

var _ configmap.Getter = optionsGetter{}

// optionsGetter is a configmap.Getter returning the default value of the
// options of an rclone backend.
type optionsGetter rfs.Options

func (o optionsGetter) Get(key string) (string, bool) {
	opt := rfs.Options(o).Get(key)
	if opt == nil {
		return "", false
	}

	return opt.String(), true
}
