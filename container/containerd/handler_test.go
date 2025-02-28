// Copyright 2017 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Handler for containerd containers.
package containerd

import (
	"testing"

	"github.com/containerd/containerd/containers"
	"github.com/containerd/typeurl"
	"github.com/google/cadvisor/container"
	containerlibcontainer "github.com/google/cadvisor/container/libcontainer"
	"github.com/google/cadvisor/fs"
	info "github.com/google/cadvisor/info/v1"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/stretchr/testify/assert"
)

func init() {
	typeurl.Register(&specs.Spec{}, "types.contianerd.io/opencontainers/runtime-spec", "v1", "Spec")
}

type mockedMachineInfo struct{}

func (m *mockedMachineInfo) GetMachineInfo() (*info.MachineInfo, error) {
	return &info.MachineInfo{}, nil
}

func (m *mockedMachineInfo) GetVersionInfo() (*info.VersionInfo, error) {
	return &info.VersionInfo{}, nil
}

func TestHandler(t *testing.T) {
	as := assert.New(t)
	type testCase struct {
		client               ContainerdClient
		name                 string
		machineInfoFactory   info.MachineInfoFactory
		fsInfo               fs.FsInfo
		cgroupSubsystems     *containerlibcontainer.CgroupSubsystems
		inHostNamespace      bool
		metadataEnvAllowList []string
		includedMetrics      container.MetricSet

		hasErr         bool
		errContains    string
		checkReference *info.ContainerReference
		checkEnvVars   map[string]string
	}
	testContainers := make(map[string]*containers.Container)
	testContainer := &containers.Container{
		ID:     "40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
		Labels: map[string]string{"io.cri-containerd.kind": "sandbox"},
	}
	spec := &specs.Spec{Root: &specs.Root{Path: "/test/"}, Process: &specs.Process{Env: []string{"TEST_REGION=FRA", "TEST_ZONE=A", "HELLO=WORLD"}}}
	testContainer.Spec, _ = typeurl.MarshalAny(spec)
	testContainers["40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"] = testContainer
	for _, ts := range []testCase{
		{
			mockcontainerdClient(nil, nil),
			"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
			nil,
			nil,
			&containerlibcontainer.CgroupSubsystems{},
			false,
			nil,
			nil,
			true,
			"unable to find container \"40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9\"",
			nil,
			nil,
		},
		{
			mockcontainerdClient(testContainers, nil),
			"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
			&mockedMachineInfo{},
			nil,
			&containerlibcontainer.CgroupSubsystems{},
			false,
			nil,
			nil,
			false,
			"",
			&info.ContainerReference{
				Id:        "40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
				Name:      "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
				Aliases:   []string{"40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9", "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"},
				Namespace: k8sContainerdNamespace,
			},
			map[string]string{},
		},
		{
			mockcontainerdClient(testContainers, nil),
			"/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
			&mockedMachineInfo{},
			nil,
			&containerlibcontainer.CgroupSubsystems{},
			false,
			[]string{"TEST"},
			nil,
			false,
			"",
			&info.ContainerReference{
				Id:        "40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
				Name:      "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9",
				Aliases:   []string{"40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9", "/kubepods/pod068e8fa0-9213-11e7-a01f-507b9d4141fa/40af7cdcbe507acad47a5a62025743ad3ddc6ab93b77b21363aa1c1d641047c9"},
				Namespace: k8sContainerdNamespace,
			},
			map[string]string{"TEST_REGION": "FRA", "TEST_ZONE": "A"},
		},
	} {
		handler, err := newContainerdContainerHandler(ts.client, ts.name, ts.machineInfoFactory, ts.fsInfo, ts.cgroupSubsystems, ts.inHostNamespace, ts.metadataEnvAllowList, ts.includedMetrics)
		if ts.hasErr {
			as.NotNil(err)
			if ts.errContains != "" {
				as.Contains(err.Error(), ts.errContains)
			}
		}
		if ts.checkReference != nil {
			cr, err := handler.ContainerReference()
			as.Nil(err)
			as.Equal(*ts.checkReference, cr)
		}
		if ts.checkEnvVars != nil {
			sp, err := handler.GetSpec()
			as.Nil(err)
			as.Equal(ts.checkEnvVars, sp.Envs)
		}
	}
}
