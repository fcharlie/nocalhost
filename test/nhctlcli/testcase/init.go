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

package testcase

import (
	"bufio"
	"context"
	"fmt"
	"github.com/imroc/req"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"io/ioutil"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/request"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/tools"
	"nocalhost/pkg/nocalhost-api/app/api/v1/service_account"
	"nocalhost/test/nhctlcli"
	"nocalhost/test/util"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

var StatusChan = make(chan int32, 1)

func GetVersion() (v1 string, v2 string) {
	commitId := os.Getenv(util.CommitId)
	var tags []string
	if len(os.Getenv(util.Tag)) != 0 {
		tags = strings.Split(strings.TrimSuffix(os.Getenv(util.Tag), "\n"), " ")
	}
	if commitId == "" && len(tags) == 0 {
		panic(fmt.Sprintf("test case failed, can not found any version, commit_id: %v, tag: %v", commitId, tags))
	}
	if len(tags) >= 2 {
		v1 = tags[0]
		v2 = tags[1]
	} else if len(tags) == 1 {
		v1 = tags[0]
	} else {
		v1 = commitId
	}
	log.Infof("version info, v1: %s, v2: %s", v1, v2)
	return
}

func InstallNhctl(version string) error {
	var name string
	var output string
	var needChmod bool
	if strings.Contains(runtime.GOOS, "darwin") {
		name = "nhctl-darwin-amd64"
		output = "nhctl"
		needChmod = true
	} else if strings.Contains(runtime.GOOS, "windows") {
		name = "nhctl-windows-amd64.exe"
		output = "nhctl.exe"
		needChmod = false
	} else {
		name = "nhctl-linux-amd64"
		output = "nhctl"
		needChmod = true
	}
	str := "curl --fail -s -L \"https://codingcorp-generic.pkg.coding.net/nocalhost/nhctl/%s?version=%s\" -o " + output
	cmd := exec.Command("sh", "-c", fmt.Sprintf(str, name, version))
	if err := nhctlcli.Runner.RunWithCheckResult(cmd); err != nil {
		return err
	}
	// unix and linux needs to add x permission
	if needChmod {
		cmd = exec.Command("sh", "-c", "chmod +x nhctl")
		if err := nhctlcli.Runner.RunWithCheckResult(cmd); err != nil {
			return err
		}
		cmd = exec.Command("sh", "-c", "sudo mv ./nhctl /usr/local/bin/nhctl")
		if err := nhctlcli.Runner.RunWithCheckResult(cmd); err != nil {
			return err
		}
	}
	return nil
}

func Init(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.CommandWithNamespace(context.Background(),
		"init", "nocalhost", "demo", "-p", "7000", "--force")
	log.Infof("Running command: %s", cmd.Args)
	var stdoutRead io.ReadCloser
	var err error
	if stdoutRead, err = cmd.StdoutPipe(); err != nil {
		return errors.Wrap(err, "stdout error")
	}
	if err = cmd.Start(); err != nil {
		_ = cmd.Process.Kill()
		return errors.Errorf("nhctl init error: %v", err)
	}
	go func() {
		if err = cmd.Wait(); err != nil {
			StatusChan <- 1
			return
		}
		StatusChan <- 0
	}()
	defer stdoutRead.Close()
	lineBody := bufio.NewReaderSize(stdoutRead, 1024)
	var line []byte
	var isPrefix bool
	go func() {
		for {
			line, isPrefix, err = lineBody.ReadLine()
			if err != nil && err != io.EOF && !strings.Contains(err.Error(), "closed") {
				fmt.Printf("command error: %v, log : %v", err, string(line))
				StatusChan <- 1
				break
			}
			if len(line) != 0 && !isPrefix {
				log.Info(string(line))
			}
			if strings.Contains(string(line), "Nocalhost init completed") {
				StatusChan <- 0
				break
			}
		}
	}()
	if i := <-StatusChan; i != 0 {
		return errors.New("Init nocalhost occurs error, exiting")
	}
	log.Infof("init successfully")
	return nil
}

func StatusCheck(nhctl *nhctlcli.CLI, moduleName string) error {
	retryTimes := 10
	var ok bool
	for i := 0; i < retryTimes; i++ {
		time.Sleep(time.Second * 3)
		cmd := nhctl.Command(context.Background(), "describe", "bookinfo", "-d", moduleName)
		stdout, stderr, err := nhctlcli.Runner.Run(cmd)
		if err != nil {
			log.Infof("Run command: %s, error: %v, stdout: %s, stderr: %s, retry", cmd.Args, err, stdout, stderr)
			continue
		}
		service := profile.SvcProfileV2{}
		_ = yaml.Unmarshal([]byte(stdout), &service)
		if !service.Developing {
			log.Info("test case failed, should be in developing, retry")
			continue
		}
		if !service.PortForwarded {
			log.Info("test case failed, should be in port forwarding, retry")
			continue
		}
		if !service.Syncing {
			log.Info("test case failed, should be in synchronizing, retry")
			continue
		}
		ok = true
		break
	}
	if !ok {
		return errors.New("test case failed, status check not pass")
	}
	return nil
}

func GetKubeconfig(ns, kubeconfig string) (string, error) {
	client, err := clientgoutils.NewClientGoUtils(kubeconfig, ns)
	log.Infof("kubeconfig %s", kubeconfig)
	if err != nil || client == nil {
		return "", errors.Errorf("new go client fail, or check you kubeconfig, err: %v", err)
	}
	kubectl, err := tools.CheckThirdPartyCLI()
	if err != nil {
		return "", errors.Errorf("check kubectl error, err: %v", err)
	}
	res := request.NewReq("", kubeconfig, kubectl, ns, 7000)
	res.ExposeService()
	res.Login(app.DefaultInitUserEmail, app.DefaultInitPassword)
	header := req.Header{"Accept": "application/json", "Authorization": "Bearer " + res.AuthToken}
	retryTimes := 20
	var config string
	for i := 0; i < retryTimes; i++ {
		time.Sleep(time.Second * 2)
		r, err := req.New().Get(res.BaseUrl+util.WebServerServiceAccountApi, header)
		if err != nil {
			log.Infof("get kubeconfig error, err: %v, response: %v, retrying", err, r)
			continue
		}
		re := Response{}
		err = r.ToJSON(&re)
		if re.Code != 0 || len(re.Data) == 0 || re.Data[0] == nil || re.Data[0].KubeConfig == "" {
			toString, _ := r.ToString()
			log.Infof("get kubeconfig response error, response: %v, string: %s, retrying", re, toString)
			continue
		}
		config = re.Data[0].KubeConfig
		break
	}
	if config == "" {
		return "", errors.New("Can't not get kubeconfig from webserver, please check your code")
	}
	f, _ := ioutil.TempFile("", "*newkubeconfig")
	_, _ = f.WriteString(config)
	_ = f.Sync()
	return f.Name(), nil
}

type Response struct {
	Code    int                                    `json:"code"`
	Message string                                 `json:"message"`
	Data    []*service_account.ServiceAccountModel `json:"data"`
}
