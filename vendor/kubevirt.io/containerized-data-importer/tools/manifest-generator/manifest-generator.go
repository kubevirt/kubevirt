//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package main

import (
	"flag"
	"github.com/golang/glog"
	"os"
	"text/template"
)

type data struct {
	DockerRepo string
	DockerTag  string
	Verbosity  string
	PullPolicy string
	Namespace  string
}

func main() {
	dockerRepo := flag.String("docker-repo", "", "")
	dockertag := flag.String("docker-tag", "", "")
	templFile := flag.String("template", "", "")
	verbosity := flag.String("verbosity", "1", "")
	pullPolicy := flag.String("pull-policy", "", "")
	namespace := flag.String("namespace", "", "")
	flag.Parse()

	data := &data{
		Verbosity:  *verbosity,
		DockerRepo: *dockerRepo,
		DockerTag:  *dockertag,
		PullPolicy: *pullPolicy,
		Namespace:  *namespace,
	}

	file, err := os.OpenFile(*templFile, os.O_RDONLY, 0)
	if err != nil {
		glog.Fatalf("Failed to open file %s: %v\n", *templFile, err)
	}
	defer file.Close()

	tmpl := template.Must(template.ParseFiles(*templFile))
	err = tmpl.Execute(os.Stdout, data)
	if err != nil {
		glog.Fatalf("Error executing template: %v\n", err)
	}
}
