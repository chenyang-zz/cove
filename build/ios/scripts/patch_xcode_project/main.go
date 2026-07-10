// Command patch_xcode_project applies the iOS build settings that Wails' alpha
// Xcode template currently omits for a Go c-archive application.
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const objcLinkerFlag = `OTHER_LDFLAGS = "-ObjC";`

var (
	bundleNamePattern       = regexp.MustCompile(`(?s)<key>CFBundleName</key>\s*<string>([^<]+)</string>`)
	bundleExecutablePattern = regexp.MustCompile(`(?s)(<key>CFBundleExecutable</key>\s*<string>)[^<]*(</string>)`)
)

func main() {
	projectPath := flag.String("project", "", "path to project.pbxproj")
	plistPath := flag.String("plist", "", "path to the generated Info.plist")
	flag.Parse()
	if *projectPath == "" || *plistPath == "" {
		fatalf("-project and -plist are required")
	}

	patchProject(*projectPath)
	patchPlist(*plistPath)
}

func patchProject(path string) {
	contents, err := os.ReadFile(path)
	if err != nil {
		fatalf("read project: %v", err)
	}
	project := string(contents)
	if !strings.Contains(project, objcLinkerFlag) {
		if !strings.Contains(project, "CODE_SIGNING_ALLOWED = NO;") {
			fatalf("unexpected Xcode signing template")
		}
		project = strings.ReplaceAll(project, "CODE_SIGNING_ALLOWED = NO;", "CODE_SIGNING_ALLOWED = YES;\n\t\t\t\t"+objcLinkerFlag)
	}
	if err := os.WriteFile(path, []byte(project), 0o644); err != nil {
		fatalf("write project: %v", err)
	}
}

func patchPlist(path string) {
	contents, err := os.ReadFile(path)
	if err != nil {
		fatalf("read Info.plist: %v", err)
	}
	plist := string(contents)
	nameMatch := bundleNamePattern.FindStringSubmatch(plist)
	if len(nameMatch) != 2 || nameMatch[1] == "" {
		fatalf("could not determine product name from Info.plist")
	}
	if !bundleExecutablePattern.MatchString(plist) {
		fatalf("Info.plist is missing CFBundleExecutable")
	}
	plist = bundleExecutablePattern.ReplaceAllString(plist, "${1}"+nameMatch[1]+"${2}")
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		fatalf("write Info.plist: %v", err)
	}
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "patch iOS Xcode project: "+format+"\n", args...)
	os.Exit(1)
}
