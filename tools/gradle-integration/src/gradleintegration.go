package gradleintegration

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config describes the project paths synchronized during devenv shell entry.
type Config struct {
	ProjectRoot         string
	AndroidHome         string
	LocalPropertiesPath string
	SDKLinkPath         string
}

var sdkDirProperty = regexp.MustCompile(`^[ \t\f]*sdk[.]dir(?:[ \t\f]*[=:]|[ \t\f]+|$)`)

// Sync refreshes the stable SDK symlink and the sdk.dir property.
func Sync(config Config) error {
	if config.LocalPropertiesPath == "" {
		return errors.New("localPropertiesPath must not be empty")
	}
	if config.SDKLinkPath == "" {
		return errors.New("sdkLinkPath must not be empty")
	}
	if strings.ContainsAny(config.LocalPropertiesPath+config.SDKLinkPath, "\r\n") {
		return errors.New("configured paths must not contain newline characters")
	}
	if config.AndroidHome == "" {
		return errors.New("ANDROID_HOME is not set; enter the devenv shell with androidSdk.enable = true")
	}

	projectRoot, err := resolveProjectRoot(config.ProjectRoot)
	if err != nil {
		return err
	}
	localProperties := resolvePath(projectRoot, config.LocalPropertiesPath)
	sdkLink := resolvePath(projectRoot, config.SDKLinkPath)

	if err := syncSDKLink(sdkLink, config.AndroidHome); err != nil {
		return err
	}
	if err := syncLocalProperties(localProperties, sdkLink); err != nil {
		return err
	}
	return nil
}

func resolveProjectRoot(root string) (string, error) {
	if root == "" {
		root = "."
	}
	if !filepath.IsAbs(root) {
		absoluteRoot, err := filepath.Abs(root)
		if err != nil {
			return "", fmt.Errorf("resolve project root %q: %w", root, err)
		}
		root = absoluteRoot
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("project root does not exist or is not accessible: %s: %w", root, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project root is not a directory: %s", root)
	}
	return root, nil
}

func resolvePath(projectRoot, configuredPath string) string {
	if filepath.IsAbs(configuredPath) {
		return configuredPath
	}
	return filepath.Join(projectRoot, configuredPath)
}

func syncSDKLink(linkPath, androidHome string) error {
	parent := filepath.Dir(linkPath)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("could not create the SDK link parent directory %s: %w", parent, err)
	}

	info, err := os.Lstat(linkPath)
	switch {
	case err == nil && info.Mode()&os.ModeSymlink == 0:
		return fmt.Errorf("refusing to replace %s because it exists and is not a symlink; remove it or configure androidSdk.gradleIntegration.sdkLinkPath differently", linkPath)
	case err == nil:
		currentTarget, readErr := os.Readlink(linkPath)
		if readErr != nil {
			return fmt.Errorf("could not read the existing SDK symlink %s: %w", linkPath, readErr)
		}
		if currentTarget == androidHome {
			return nil
		}
		if replaceErr := replaceSymlink(linkPath, androidHome); replaceErr != nil {
			return fmt.Errorf("could not refresh the SDK symlink %s: %w", linkPath, replaceErr)
		}
		return nil
	case !errors.Is(err, os.ErrNotExist):
		return fmt.Errorf("could not inspect the SDK link path %s: %w", linkPath, err)
	default:
		if symlinkErr := os.Symlink(androidHome, linkPath); symlinkErr != nil {
			return fmt.Errorf("could not create the SDK symlink %s: %w", linkPath, symlinkErr)
		}
		return nil
	}
}

func replaceSymlink(linkPath, target string) error {
	parent := filepath.Dir(linkPath)
	temporaryFile, err := os.CreateTemp(parent, ".devenv-android-sdk-link-*")
	if err != nil {
		return err
	}
	temporaryPath := temporaryFile.Name()
	if closeErr := temporaryFile.Close(); closeErr != nil {
		_ = os.Remove(temporaryPath)
		return closeErr
	}
	if removeErr := os.Remove(temporaryPath); removeErr != nil {
		return removeErr
	}
	defer os.Remove(temporaryPath) // Best-effort cleanup after failures.

	if err := os.Symlink(target, temporaryPath); err != nil {
		return err
	}
	return os.Rename(temporaryPath, linkPath)
}

func syncLocalProperties(path, sdkLink string) error {
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("could not create the local.properties parent directory %s: %w", parent, err)
	}

	info, err := os.Lstat(path)
	exists := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("could not inspect local.properties at %s: %w", path, err)
	}

	isSymlink := false
	if exists {
		isSymlink = info.Mode()&os.ModeSymlink != 0
		if isSymlink {
			targetInfo, statErr := os.Stat(path)
			if statErr != nil {
				if errors.Is(statErr, os.ErrNotExist) {
					return fmt.Errorf("%s is a broken symlink; repair its target or remove the symlink before entering the devenv shell", path)
				}
				return fmt.Errorf("could not inspect the local.properties symlink target at %s: %w", path, statErr)
			}
			if !targetInfo.Mode().IsRegular() {
				return fmt.Errorf("%s points to an object that is not a regular file", path)
			}
		} else if !info.Mode().IsRegular() {
			return fmt.Errorf("%s exists but is not a regular file (or a symlink to one)", path)
		}
	}

	var current []byte
	if exists {
		current, err = os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("%s is not readable: %w", path, err)
		}
	}

	desired := synchronizeSDKDir(current, sdkLink)
	if exists && bytes.Equal(current, desired) {
		return nil
	}

	if exists {
		probe, openErr := os.OpenFile(path, os.O_WRONLY, 0)
		if openErr != nil {
			return fmt.Errorf("%s is not writable; make its target writable before entering the devenv shell: %w", path, openErr)
		}
		if closeErr := probe.Close(); closeErr != nil {
			return fmt.Errorf("could not close %s after checking writability: %w", path, closeErr)
		}
	}

	if exists && !isSymlink {
		if err := writeRegularFileAtomically(path, desired, info.Mode().Perm()); err != nil {
			return fmt.Errorf("could not atomically update %s: %w", path, err)
		}
		return nil
	}

	flags := os.O_WRONLY | os.O_TRUNC
	if !exists {
		flags |= os.O_CREATE | os.O_EXCL
	}
	if err := writeFileContents(path, desired, flags, 0o666); err != nil {
		return fmt.Errorf("could not write %s; check that the file or symlink target is writable: %w", path, err)
	}
	return nil
}

func writeRegularFileAtomically(path string, content []byte, mode os.FileMode) error {
	temporaryFile, err := os.CreateTemp(filepath.Dir(path), ".local.properties-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath) // Best-effort cleanup after failures.

	if err := temporaryFile.Chmod(mode); err != nil {
		_ = temporaryFile.Close()
		return fmt.Errorf("preserve file permissions: %w", err)
	}
	if err := writeAndClose(temporaryFile, content); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace file: %w", err)
	}
	return nil
}

func writeFileContents(path string, content []byte, flags int, mode os.FileMode) error {
	file, err := os.OpenFile(path, flags, mode)
	if err != nil {
		return err
	}
	return writeAndClose(file, content)
}

func writeAndClose(file *os.File, content []byte) error {
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		return fmt.Errorf("write content: %w", err)
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("flush content: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}

func synchronizeSDKDir(current []byte, sdkLink string) []byte {
	entry := []byte("sdk.dir=" + strings.ReplaceAll(sdkLink, `\`, `\\`))
	result := make([]byte, 0, len(current)+len(entry)+1)
	found := false

	for offset := 0; offset < len(current); {
		relativeEnd := bytes.IndexByte(current[offset:], '\n')
		end := len(current)
		if relativeEnd >= 0 {
			end = offset + relativeEnd + 1
		}
		line := current[offset:end]
		content := bytes.TrimSuffix(line, []byte("\n"))
		content = bytes.TrimSuffix(content, []byte("\r"))

		if sdkDirProperty.Match(content) {
			if !found {
				result = append(result, entry...)
				if bytes.HasSuffix(line, []byte("\r\n")) {
					result = append(result, '\r', '\n')
				} else if bytes.HasSuffix(line, []byte("\n")) {
					result = append(result, '\n')
				}
				found = true
			}
		} else {
			result = append(result, line...)
		}
		offset = end
	}

	if !found {
		if len(result) > 0 && result[len(result)-1] != '\n' {
			result = append(result, '\n')
		}
		result = append(result, entry...)
		result = append(result, '\n')
	}
	return result
}
