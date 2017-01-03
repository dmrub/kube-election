/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	election "k8s.io/contrib/election/lib"

	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	kubectl_util "k8s.io/kubernetes/pkg/kubectl/cmd/util"
)

var (
	flags = flag.NewFlagSet(
		`elector --election=<name>`,
		flag.ExitOnError)
	name         = flags.String("election", "", "The name of the election")
	id           = flags.String("id", "", "The id of this participant")
	namespace    = flags.String("election-namespace", api.NamespaceDefault, "The Kubernetes namespace for this election")
	ttl          = flags.Duration("ttl", 10*time.Second, "The TTL for this election")
	inCluster    = flags.Bool("use-cluster-credentials", false, "Should this request use cluster credentials?")
	resolvePodID = flags.Bool("resolve-pod-ip", false, "Use participant ID as Pod name and resolve Pod IP of the elected leader")
	addr         = flags.String("http", "", "If non-empty, stand up a simple webserver that reports the leader state")

	mu     sync.Mutex
	leader = &LeaderData{}
)

func makeClient() (*client.Client, error) {
	var cfg *restclient.Config
	var err error

	if *inCluster {
		if cfg, err = restclient.InClusterConfig(); err != nil {
			return nil, err
		}
	} else {
		clientConfig := kubectl_util.DefaultClientConfig(flags)
		if cfg, err = clientConfig.ClientConfig(); err != nil {
			return nil, err
		}
	}
	return client.New(cfg)
}

// LeaderData represents information about the current leader
type LeaderData struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	PodIP     string `json:"podIP,omitempty"`
}

func webHandler(res http.ResponseWriter, req *http.Request) {
	mu.Lock()
	data, err := json.Marshal(leader)
	mu.Unlock()
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write([]byte(err.Error()))
		return
	}
	res.WriteHeader(http.StatusOK)
	res.Write(data)
}

func validateFlags() {
	if len(*id) == 0 {
		glog.Fatal("--id cannot be empty")
	}
	if len(*name) == 0 {
		glog.Fatal("--election cannot be empty")
	}
}

func main() {
	flags.Parse(os.Args)
	validateFlags()
	leader.Namespace = *namespace

	kubeClient, err := makeClient()
	if err != nil {
		glog.Fatalf("error connecting to the client: %v", err)
	}

	fn := func(str string) {
		fmt.Printf("%s is the leader\n", str)
		var podIP string
		if *resolvePodID {
			pod, err := kubeClient.Pods(*namespace).Get(str)
			if err != nil {
				glog.Errorf("failed to get Pod %s: %v", str, err)
			} else {
				podIP = pod.Status.PodIP
			}
		}

		mu.Lock()
		leader.Name = str
		leader.PodIP = podIP
		mu.Unlock()
	}

	e, err := election.NewElection(*name, *id, *namespace, *ttl, fn, kubeClient)
	if err != nil {
		glog.Fatalf("failed to create election: %v", err)
	}
	go election.RunElection(e)

	if len(*addr) > 0 {
		http.HandleFunc("/", webHandler)
		http.ListenAndServe(*addr, nil)
	} else {
		select {}
	}
}
