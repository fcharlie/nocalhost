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

package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/pkg/nhctl/log"
)

var configKey string
var configVal string

const (
	imageKey  = "image"
	gitUrlKey = "gitUrl"
)

func init() {
	profileSetCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s workload name")
	profileSetCmd.Flags().StringVarP(&serviceType, "type", "t", "deployment", "specify service type")
	profileSetCmd.Flags().StringVarP(&container, "container", "c", "", "container name of pod")
	profileSetCmd.Flags().StringVarP(&configKey, "key", "k", "", "key of dev config")
	profileSetCmd.Flags().StringVarP(&configVal, "value", "v", "", "value of dev config")
	profileCmd.AddCommand(profileSetCmd)
}

var profileSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a config item of Profile",
	Long:  `Set a config item of Profile`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		initAppAndCheckIfSvcExist(args[0], deployment, serviceType)
		if configKey == "" {
			log.Fatal("--key must be specified")
		}
		if configVal == "" {
			log.Fatal("--value must be specified")
		}
		if container == "" {
			log.Fatal("--container must be specified")
		}

		supportedConfigKey := []string{imageKey, gitUrlKey}
		if !stringSliceContains(supportedConfigKey, configKey) {
			log.Fatalf("Config key %s is unsupported", configKey)
		}

		nocalhostSvc.UpdateSvcProfile(func(v2 *profile.SvcProfileV2) error {
			var defaultContainerConfig, targetContainerConfig *profile.ContainerConfig
			for _, c := range v2.ContainerConfigs {
				if c.Name == "" {
					defaultContainerConfig = c
				} else if c.Name == container {
					targetContainerConfig = c
					break
				}
			}
			if targetContainerConfig == nil && defaultContainerConfig != nil {
				defaultContainerConfig.Name = container
				targetContainerConfig = defaultContainerConfig
			}

			if targetContainerConfig != nil {
				if targetContainerConfig.Dev == nil {
					targetContainerConfig.Dev = &profile.ContainerDevConfig{}
				}
				if configKey == imageKey {
					targetContainerConfig.Dev.Image = configVal
				} else if configKey == gitUrlKey {
					targetContainerConfig.Dev.GitUrl = configVal
				}
				return nil
			}
			// Create one
			targetContainerConfig = &profile.ContainerConfig{Dev: &profile.ContainerDevConfig{}, Name: container}
			switch configKey {
			case imageKey:
				targetContainerConfig.Dev.Image = configVal
			case gitUrlKey:
				targetContainerConfig.Dev.GitUrl = configVal
			}
			v2.ContainerConfigs = append(v2.ContainerConfigs, targetContainerConfig)
			return nil
		})
	},
}

func stringSliceContains(ss []string, item string) bool {
	for _, s := range ss {
		if s == item {
			return true
		}
	}
	return false
}
