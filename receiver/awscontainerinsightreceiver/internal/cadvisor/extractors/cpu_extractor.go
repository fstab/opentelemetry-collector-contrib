// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extractors // import "github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awscontainerinsightreceiver/internal/cadvisor/extractors"

import (
	cInfo "github.com/google/cadvisor/info/v1"
	"go.uber.org/zap"

	ci "github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/containerinsight"
	awsmetrics "github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/metrics"
)

const (
	decimalToMillicores = 1000
)

type CPUMetricExtractor struct {
	logger         *zap.Logger
	rateCalculator awsmetrics.MetricCalculator
}

func (c *CPUMetricExtractor) HasValue(info *cInfo.ContainerInfo) bool {
	return info.Spec.HasCpu
}

func (c *CPUMetricExtractor) GetValue(info *cInfo.ContainerInfo, mInfo CPUMemInfoProvider, containerType string) []*CAdvisorMetric {
	var metrics []*CAdvisorMetric
	if info.Spec.Labels[containerNameLable] == infraContainerName {
		return metrics
	}

	// When there is more than one stats point, always use the last one
	curStats := GetStats(info)
	metric := newCadvisorMetric(containerType, c.logger)
	multiplier := float64(decimalToMillicores)
	assignRateValueToField(&c.rateCalculator, metric.fields, ci.MetricName(containerType, ci.CPUTotal), info.Name, float64(curStats.Cpu.Usage.Total), curStats.Timestamp, multiplier)
	assignRateValueToField(&c.rateCalculator, metric.fields, ci.MetricName(containerType, ci.CPUUser), info.Name, float64(curStats.Cpu.Usage.User), curStats.Timestamp, multiplier)
	assignRateValueToField(&c.rateCalculator, metric.fields, ci.MetricName(containerType, ci.CPUSystem), info.Name, float64(curStats.Cpu.Usage.System), curStats.Timestamp, multiplier)

	numCores := mInfo.GetNumCores()
	if metric.fields[ci.MetricName(containerType, ci.CPUTotal)] != nil && numCores != 0 {
		metric.fields[ci.MetricName(containerType, ci.CPUUtilization)] = metric.fields[ci.MetricName(containerType, ci.CPUTotal)].(float64) / float64(numCores*decimalToMillicores) * 100
	}

	if containerType == ci.TypeNode || containerType == ci.TypeInstance {
		metric.fields[ci.MetricName(containerType, ci.CPULimit)] = numCores * decimalToMillicores
	}

	metrics = append(metrics, metric)
	return metrics
}

func NewCPUMetricExtractor(logger *zap.Logger) *CPUMetricExtractor {
	return &CPUMetricExtractor{
		logger:         logger,
		rateCalculator: newFloat64RateCalculator(),
	}
}