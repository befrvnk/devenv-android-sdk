package main

import (
	"fmt"
	"os"

	gradleintegration "github.com/befrvnk/devenv-android-sdk/tools/gradle-integration"
)

func main() {
	if len(os.Args) != 3 {
		fail("expected local.properties path and SDK link path arguments")
	}

	projectRoot := os.Getenv("DEVENV_ROOT")
	if projectRoot == "" {
		var err error
		projectRoot, err = os.Getwd()
		if err != nil {
			fail(fmt.Sprintf("could not resolve the current directory: %v", err))
		}
	}

	err := gradleintegration.Sync(gradleintegration.Config{
		ProjectRoot:         projectRoot,
		AndroidHome:         os.Getenv("ANDROID_HOME"),
		LocalPropertiesPath: os.Args[1],
		SDKLinkPath:         os.Args[2],
	})
	if err != nil {
		fail(err.Error())
	}
}

func fail(message string) {
	fmt.Fprintf(os.Stderr, "devenv Android SDK Gradle integration: %s\n", message)
	os.Exit(1)
}
