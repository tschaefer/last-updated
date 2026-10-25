/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package version

import (
	"fmt"
	"os"
)

// GitCommit, Version variables set while build process.
var (
	GitCommit, Version string
)

// Release returns the tagged version or "dev" if no tag is available.
func Release() string {
	if Version == "" {
		Version = "dev"
	}

	return Version
}

// Release returns the current commit hash or empty if not available.
func Commit() string {
	return GitCommit
}

// Banner returns the app label.
func Banner() string {
	return `
 _           _                        _       _           _
| | __ _ ___| |_      _   _ _ __   __| | __ _| |_ ___  __| |
| |/ _' / __| __|____| | | | '_ \ / _' |/ _' | __/ _ \/ _' |
| | (_| \__ \ ||_____| |_| | |_) | (_| | (_| | ||  __/ (_| |
|_|\__,_|___/\__|     \__,_| .__/ \__,_|\__,_|\__\___|\__,_|
                           |_|
`
}

// Print prints the colored banner and all version information. The
// environment variable NO_COLOR is respected.
func Print() {
	no_color := os.Getenv("NO_COLOR")
	if no_color != "" {
		fmt.Printf("%s\n", Banner())
	} else {
		fmt.Printf("\033[34m%s\033[0m\n", Banner())
	}
	fmt.Printf("Release: %s\n", Release())
	fmt.Printf("Commit:  %s\n", Commit())
}
