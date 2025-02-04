/*
Copyright 2019 The Skaffold Authors

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

package runner

import (
	"context"
	"io"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build/cache"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/label"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/status"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/filemon"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/graph"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/sync"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/tag"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/test"
)

const (
	remoteDigestSource = "remote"
	noneDigestSource   = "none"
)

// Runner is responsible for running the skaffold build, test and deploy config.
type Runner interface {
	Apply(context.Context, io.Writer) error
	ApplyDefaultRepo(tag string) (string, error)
	Build(context.Context, io.Writer, []*latest.Artifact) ([]graph.Artifact, error)
	Cleanup(context.Context, io.Writer) error
	Dev(context.Context, io.Writer, []*latest.Artifact) error
	DeployAndLog(context.Context, io.Writer, []graph.Artifact) error
	GeneratePipeline(context.Context, io.Writer, []*latest.SkaffoldConfig, []string, string) error
	HasBuilt() bool
	HasDeployed() bool
	Prune(context.Context, io.Writer) error
	Render(context.Context, io.Writer, []graph.Artifact, bool, string) error
	Test(context.Context, io.Writer, []graph.Artifact) error
}

// SkaffoldRunner is responsible for running the skaffold build, test and deploy config.
type SkaffoldRunner struct {
	builder  build.Builder
	deployer deploy.Deployer
	tester   test.Tester
	tagger   tag.Tagger
	syncer   sync.Syncer
	monitor  filemon.Monitor
	listener Listener

	kubectlCLI         *kubectl.CLI
	cache              cache.Cache
	changeSet          changeSet
	runCtx             *runcontext.RunContext
	labeller           *label.DefaultLabeller
	builds             []graph.Artifact
	artifactStore      build.ArtifactStore
	sourceDependencies graph.TransitiveSourceDependenciesCache
	// podSelector is used to determine relevant pods for logging and portForwarding
	podSelector *kubernetes.ImageList

	isLocalImage func(imageName string) (bool, error)
	hasBuilt     bool
	hasDeployed  bool
	intents      *intents
	devIteration int
}

// for testing
var (
	newStatusCheck = status.NewStatusChecker
)

// HasDeployed returns true if this runner has deployed something.
func (r *SkaffoldRunner) HasDeployed() bool {
	return r.hasDeployed
}

// HasBuilt returns true if this runner has built something.
func (r *SkaffoldRunner) HasBuilt() bool {
	return r.hasBuilt
}
