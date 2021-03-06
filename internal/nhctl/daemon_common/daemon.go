/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package daemon_common

import (
	"context"
	"github.com/pkg/errors"
	"io/ioutil"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/internal/nhctl/utils"
	"os"
	"path/filepath"
)

const (
	DefaultDaemonPort = 30123
	SudoDaemonPort    = 30124
)

var (
	Version  = "1.0"
	CommitId = ""
)

type DaemonServerInfo struct {
	Version   string
	CommitId  string
	NhctlPath string
	Upgrading bool
}

type PortForwardProfile struct {
	Cancel     context.CancelFunc `json:"-"` // For canceling a port forward
	StopCh     chan error         `json:"-"`
	NameSpace  string             `json:"nameSpace"`
	AppName    string             `json:"appName"`
	SvcName    string             `json:"svcName"`
	SvcType    string             `json:"svcType"`
	Role       string             `json:"role"`
	LocalPort  int                `json:"localPort"`
	RemotePort int                `json:"remotePort"`
}

func NewDaemonServerInfo() *DaemonServerInfo {
	return &DaemonServerInfo{Version: Version}
}

type DaemonServerStatusResponse struct {
	PortForwardList []*PortForwardProfile `json:"portForwardList"`
}

// StartDaemonServerBySubProcess
// In windows, we need to copy nhctl.exe to a tmpDir and then run daemon from tmpDir.
// Otherwise, we can not upgrade nhctl.exe when daemon is running
func StartDaemonServerBySubProcess(isSudoUser bool) error {
	var (
		nhctlPath string
		err       error
	)
	if utils.IsWindows() {
		if nhctlPath, err = CopyNhctlBinaryToTmpDir(os.Args[0]); err != nil {
			return err
		}
	} else {
		if nhctlPath, err = utils.GetNhctlPath(); err != nil {
			return err
		}
	}
	daemonArgs := []string{nhctlPath, "daemon", "start"}
	if isSudoUser {
		daemonArgs = append(daemonArgs, "--sudo", "true")
	}
	return daemon.RunSubProcess(daemonArgs, nil, false)
}

// CopyNhctlBinaryToTmpDir
// Copy nhctl binary to a tmpDir and return the path of nhctl in tmpDir
func CopyNhctlBinaryToTmpDir(nhctlPath string) (string, error) {
	daemonDir, err := ioutil.TempDir("", "nhctl-daemon")
	if err != nil {
		return "", errors.Wrap(err, "")
	}
	// cp nhctl to daemonDir
	if err = utils.CopyFile(nhctlPath, filepath.Join(daemonDir, utils.GetNhctlBinName())); err != nil {
		return "", errors.Wrap(err, "")
	}
	return filepath.Join(daemonDir, utils.GetNhctlBinName()), nil
}
