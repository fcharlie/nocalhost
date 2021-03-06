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
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/fp"
	"nocalhost/test/nhctlcli"
	"strings"
	"time"
)

type containerImage struct {
	Image string `json:"image"`
}

func ProfileGetUbuntuWithJson(nhctl *nhctlcli.CLI) error {
	return profileGetWithJson(nhctl, "ubuntu", "any")
}

func ProfileGetDetailsWithoutJson(nhctl *nhctlcli.CLI) error {
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
	cmd := nhctl.Command(
		ctx, "profile",
		"get", "bookinfo",
		"-d", "details", "-t", "deployment", "--container", "details", "--key", "image",
	)
	stdout, stderr, err := nhctlcli.Runner.Run(cmd)
	if err != nil {
		return err
	}
	if stderr != "" {
		return errors.New(stderr)
	}
	if stdout != "" {
		return errors.New(fmt.Sprintf("output: %s, profile get should be nil", stdout))
	}
	return nil
}

func ProfileSetDetails(nhctl *nhctlcli.CLI) error {
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
	cmd := nhctl.Command(
		ctx, "profile",
		"set", "bookinfo",
		"-d", "details", "-t", "deployment", "--container", "details", "--key", "image",
		"--value", "helloWorld",
	)
	err := nhctlcli.Runner.RunWithCheckResult(cmd)
	if err != nil {
		return err
	}

	return profileGetWithJson(nhctl, "details", "helloWorld")
}

func ApplyCmForConfig(nhctl *nhctlcli.CLI, filePath *fp.FilePathEnhance) error {
	cmd := nhctl.Command(
		context.TODO(), "apply", "bookinfo", filePath.Abs(),
	)

	return nhctlcli.Runner.RunSimple(
		cmd, func(s string) error {
			return nil
		},
	)
}

func ValidateImage(nhctl *nhctlcli.CLI, svcName string, svcType string, expectContain string) error {
	cmd := nhctl.Command(
		context.TODO(), "profile", "get", "bookinfo", "-d", svcName, "-t", svcType,
		"--container", "xx",
		"--key", "image",
	)

	return nhctlcli.Runner.RunSimple(
		cmd, func(s string) error {
			if !strings.Contains(s, expectContain) {
				return errors.New(
					fmt.Sprintf(
						"profile not expected!! shold contain %s, now image %s", expectContain, s,
					),
				)
			}
			return nil
		},
	)
}

func ConfigReload(nhctl *nhctlcli.CLI) error {
	cmd := nhctl.Command(
		context.TODO(), "config",
		"reload", "bookinfo",
	)

	return nhctlcli.Runner.RunSimple(
		cmd, func(s string) error {
			return nil
		},
	)
}

func DeAssociate(nhctl *nhctlcli.CLI, svcName string, svcType string) error {
	cmd := nhctl.Command(
		context.TODO(), "dev",
		"associate", "bookinfo",
		"-d", svcName, "-t", svcType, "--de-associate",
	)

	return nhctlcli.Runner.RunSimple(
		cmd, func(s string) error {
			return nil
		},
	)
}

func Associate(nhctl *nhctlcli.CLI, svcName string, svcType string, dir *fp.FilePathEnhance) error {
	cmd := nhctl.Command(
		context.TODO(), "dev",
		"associate", "bookinfo",
		"-d", svcName, "-t", svcType, "--associate", dir.Abs(),
	)

	return nhctlcli.Runner.RunSimple(
		cmd, func(s string) error {
			return nil
		},
	)
}

func profileGetWithJson(nhctl *nhctlcli.CLI, container string, image string) error {
	tmp := &containerImage{}
	ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
	cmd := nhctl.Command(
		ctx, "profile",
		"get", "bookinfo",
		"-d", "details", "-t", "deployment", "--container", container, "--key", "image",
	)
	stdout, stderr, err := nhctlcli.Runner.Run(cmd)
	if err != nil {
		return err
	}
	if stderr != "" {
		return errors.New(stderr)
	}
	if stdout == "" {
		return errors.New("profile get should be a json")
	}

	err = json.Unmarshal([]byte(stdout), tmp)
	if err != nil {
		return err
	}

	if image == "any" {
		if tmp.Image == "" {
			return errors.New("image of dev container config should not be nil")
		}
	} else {
		if tmp.Image != image {
			fmt.Printf("Image is %s\n", tmp.Image)
			return errors.New(fmt.Sprintf("image of dev container config should be %s", image))
		}
	}
	return nil
}
