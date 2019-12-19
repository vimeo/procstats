package cgrouplimits

import "errors"

// ErrCGroupsNotSupported is returned on non-linux systems (outside virtualized
// docker containers)
var ErrCGroupsNotSupported = errors.New(
	"this platform does not support cgroups")

// ErrUnimplementedPlatform is returned on systems for which usage/limits
// querying has not been implemented.
var ErrUnimplementedPlatform = errors.New("support for this platform is unimplmented")
