package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stoewer/go-strcase"
)

const postContentPath = "docs/hugo-tania/site/content/post"

func main() {
	files, err := ioutil.ReadDir(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	workDir, _ := os.Getwd()

	docPath := filepath.Join(workDir, "docs")

	for _, f := range files {
		if f.IsDir() && !strings.HasPrefix(f.Name(), ".") {
			serviceDir := filepath.Join(workDir, f.Name())
			serviceFiles, err := ioutil.ReadDir(serviceDir)
			if err != nil {
				fmt.Println("Failed to read service dir", err)
				os.Exit(1)
			}
			skip := false
			for _, serviceFile := range serviceFiles {
				if serviceFile.Name() == "skip" {
					skip = true
				}
			}
			if skip {
				continue
			}

			fmt.Println("Processing folder", serviceDir)
			makeProto := exec.Command("make", "proto")
			makeProto.Dir = serviceDir
			fmt.Println(serviceDir)
			outp, err := makeProto.CombinedOutput()
			if err != nil {
				fmt.Println("Failed to make proto", string(outp))
				os.Exit(1)
			}
			serviceName := f.Name()
			dat, err := ioutil.ReadFile(filepath.Join(serviceDir, "README.md"))
			if err != nil {
				fmt.Println("Failed to read readme", string(outp))
				os.Exit(1)
			}

			contentDir := filepath.Join(workDir, postContentPath)
			err = os.MkdirAll(contentDir, 0777)
			if err != nil {
				fmt.Println("Failed to create content dir", string(outp))
				os.Exit(1)
			}

			contentFile := filepath.Join(workDir, postContentPath, serviceName+".md")
			err = ioutil.WriteFile(contentFile, append([]byte("---\ntitle: "+serviceName+"\n---\n"), dat...), 0777)
			if err != nil {
				fmt.Printf("Failed to write post content to %v:\n%v\n", contentFile, string(outp))
				os.Exit(1)
			}

			apiJSON := filepath.Join(serviceDir, "api-"+serviceName+".json")
			js, err := ioutil.ReadFile(apiJSON)
			if err != nil {
				apiJSON := filepath.Join(serviceDir, "api-protobuf.json")
				js, err = ioutil.ReadFile(apiJSON)
				if err != nil {
					fmt.Println("Failed to read json spec", err)
					os.Exit(1)
				}
			}
			spec := &openapi3.Swagger{}
			err = json.Unmarshal(js, &spec)
			if err != nil {
				fmt.Println("Failed to unmarshal", err)
				os.Exit(1)
			}
			err = saveSpec(contentFile, spec)
			if err != nil {
				fmt.Println("Failed to save to spec file", err)
				os.Exit(1)
			}

			openAPIDir := filepath.Join(docPath, serviceName, "api")
			err = os.MkdirAll(openAPIDir, 0777)
			if err != nil {
				fmt.Println("Failed to create api folder", string(outp))
				os.Exit(1)
			}

			err = CopyFile(filepath.Join(serviceDir, "redoc-static.html"), filepath.Join(docPath, serviceName, "api", "index.html"))
			if err != nil {
				fmt.Println("Failed to copy redoc", string(outp))
				os.Exit(1)
			}

			cmd := exec.Command("hugo", "-D", "-d", "../../")
			cmd.Dir = filepath.Join(docPath, "hugo-tania", "site")
			outp, err = cmd.CombinedOutput()
			if err != nil {
				fmt.Println("Build hugo site", string(outp))
				os.Exit(1)
			}
		}
	}
}

func saveSpec(filepath string, spec *openapi3.Swagger) error {
	fi, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}
	//spec.Paths[0].Parameters[0].MarshalJSON()
	//spec.Paths[0].Options
	//spec.Paths[0].Post.RequestBody.
	tmpl, err := template.New("test").Funcs(template.FuncMap{
		"params": func(p openapi3.Parameters) string {
			ls := ""
			for _, v := range p {
				//if v.Value.In == "body" {
				bs, _ := v.MarshalJSON()
				ls += string(bs) + ", "
				//}
			}
			return ls
		},
		// @todo should take SpecRef here not RequestBodyRef
		"schemaJSON": func(ref string) string {
			for k, v := range spec.Components.Schemas {
				// ie. #/components/requestBodies/PostsSaveRequest contains
				// SaveRequest, can't see any other way to correlate
				if strings.HasSuffix(ref, k) {
					bs, _ := json.MarshalIndent(schemaToMap(v, spec.Components.Schemas), "", "  ")
					return string(bs)
				}
			}

			return "Schema related to " + ref + " not found"

		},
		"schemaDescription": func(ref string) string {
			for k, v := range spec.Components.Schemas {
				// ie. #/components/requestBodies/PostsSaveRequest contains
				// SaveRequest, can't see any other way to correlate
				if strings.HasSuffix(ref, k) {
					return v.Value.Description
				}
			}

			return "Schema related to " + ref + " not found"

		},
		// turn chat/Chat/History
		// to Chat History
		"titleize": func(s string) string {
			parts := strings.Split(s, "/")
			if len(parts) > 2 {
				return strings.Join(parts[2:], " ")
			}
			return strings.Join(parts, " ")
		},
		"firstResponseRef": func(rs openapi3.Responses) string {
			return rs.Get(200).Ref
		},
	}).Parse(templ)
	if err != nil {
		panic(err)
	}
	return tmpl.Execute(fi, spec)
}

func schemaToMap(spec *openapi3.SchemaRef, schemas map[string]*openapi3.SchemaRef) map[string]interface{} {
	var recurse func(props map[string]*openapi3.SchemaRef) map[string]interface{}
	recurse = func(props map[string]*openapi3.SchemaRef) map[string]interface{} {
		ret := map[string]interface{}{}
		for k, v := range props {
			k = strcase.SnakeCase(k)
			//v.Value.
			if v.Value.Type == "object" {
				// @todo identify what is a slice and what is not!
				// currently the openapi converter messes this up
				// see redoc html output
				ret[k] = []interface{}{recurse(v.Value.Properties)}
				continue
			}
			switch v.Value.Type {
			case "string":
				if len(v.Value.Description) > 0 {
					ret[k] = strings.Replace(v.Value.Description, "\n", ".", -1)
				} else {
					ret[k] = v.Value.Type
				}
			case "number":
				ret[k] = 1
			case "boolean":
				ret[k] = true
			}

		}
		return ret
	}
	return recurse(spec.Value.Properties)
}

const templ = `
## cURL

{{ range $key, $value := .Paths }}
### {{ $key | titleize }}
<!-- We use the request body description here as endpoint descriptions are not
being lifted correctly from the proto by the openapi spec generator -->
{{ $value.Post.RequestBody.Ref | schemaDescription }}
` + "```" + `shell
> curl 'https://api.m3o.com{{ $key }}' \
  -H 'micro-namespace: $yourNamespace' \
  -H 'authorization: Bearer $yourToken' \
  -d {{ $value.Post.RequestBody.Ref | schemaJSON }};
# Response
{{ $value.Post.Responses | firstResponseRef | schemaJSON }}
` + "```" + `

{{ end }}
`

// CopyFile copies a file from src to dst. If src and dst files exist, and are
// the same, then return success. Otherise, attempt to create a hard link
// between the two files. If that fail, copy the file contents from src to dst.
// from https://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file-in-golang
func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
