package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

func main() {
	if len(os.Args) < 2 {
		printUsage(os.Stderr)
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "route":
		_, err = runRouteCommand(os.Args[2:])
	case "repository":
		_, err = runRepositoryCommand(os.Args[2:])
	default:
		printUsage(os.Stderr)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runRouteCommand(args []string) (Report, error) {
	fs := flag.NewFlagSet("route", flag.ContinueOnError)
	root := fs.String("root", ".", "project root")
	noColor := fs.Bool("no-color", false, "disable colored output")
	if err := fs.Parse(args); err != nil {
		return Report{}, err
	}

	report, err := GenerateRoutes(*root)
	if err != nil {
		return report, err
	}
	printReport(os.Stdout, report, !*noColor)
	return report, nil
}

func runRepositoryCommand(args []string) (Report, error) {
	fs := flag.NewFlagSet("repository", flag.ContinueOnError)
	root := fs.String("root", ".", "project root")
	model := fs.String("model", "", "GORM model name")
	label := fs.String("label", "", "human readable model label for errors")
	scope := fs.String("scope", "", "repository user scope, format local_column:table.column:user_column")
	noColor := fs.Bool("no-color", false, "disable colored output")
	if err := fs.Parse(args); err != nil {
		return Report{}, err
	}

	report, err := GenerateRepository(RepositoryOptions{
		Root:  *root,
		Model: *model,
		Label: *label,
		Scope: *scope,
	})
	if err != nil {
		return report, err
	}
	printReport(os.Stdout, report, !*noColor)
	return report, nil
}

func printUsage(w *os.File) {
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  codegen route [-root .] [-no-color]")
	fmt.Fprintln(w, "  codegen repository -model Model [-label 名称] [-scope local_column:table.column:user_column] [-root .] [-no-color]")
}

func GenerateRoutes(root string) (Report, error) {
	report := Report{Root: root}
	routes, err := scanRoutes(root)
	if err != nil {
		return report, err
	}
	if len(routes) == 0 {
		return report, nil
	}

	handlers, err := scanHandlers(root)
	if err != nil {
		return report, err
	}
	logics, err := scanLogics(root)
	if err != nil {
		return report, err
	}
	requestDTOs, err := scanRequestDTOs(root)
	if err != nil {
		return report, err
	}

	routesByDomain := map[string][]Route{}
	for _, route := range routes {
		routesByDomain[route.Domain] = append(routesByDomain[route.Domain], route)
	}
	domains := make([]string, 0, len(routesByDomain))
	for domain := range routesByDomain {
		domains = append(domains, domain)
	}
	sort.Strings(domains)

	for _, domain := range domains {
		if err := generateHandler(root, domain, routesByDomain[domain], handlers, requestDTOs, &report); err != nil {
			return report, err
		}
		for _, route := range routesByDomain[domain] {
			key := logicKey(route.Domain, route.HandlerMethod)
			if path, ok := logics[key]; ok {
				report.Add(FileSkipped, path)
				continue
			}
			if logicFileExists(root, route) {
				report.Add(FileSkipped, logicPath(root, route))
				continue
			}
			if err := generateLogic(root, route, &report); err != nil {
				return report, err
			}
			logics[key] = logicPath(root, route)
		}
	}
	return report, nil
}
