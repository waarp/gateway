package wg

type overrideCommand struct {
	Address overrideAddressCommand `command:"address" description:"Manage net address indirections"`
}
