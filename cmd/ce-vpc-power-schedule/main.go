package main

import (
	"ce-vpc-power-schedule/internal/logging"
	"ce-vpc-power-schedule/internal/vpc"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	instanceType       = "instance"
	tagType            = "tag"
	placementGroupType = "placement_group"
	resourceGroupType  = "resource_group"
	vpcType            = "vpc"
)

type Entity struct {
	Type  string `yaml:"type"`  // instance | tag | placement_group | resource_group | vpc
	Value string `yaml:"value"` // instance name, tag name, placement group name, resource group name, or VPC name
}

type DateExclusion struct {
	Date   string `yaml:"date"`
	Annual bool   `yaml:"annual"`
}

type RangeExclusion struct {
	From   string `yaml:"from"`
	To     string `yaml:"to"`
	Annual bool   `yaml:"annual"`
}

type Exclusions struct {
	Timezone string           `yaml:"timezone"`
	Dates    []DateExclusion  `yaml:"dates"`
	Ranges   []RangeExclusion `yaml:"ranges"`
}

type Config struct {
	Entities      []Entity   `yaml:"entities"`
	Exclusions    Exclusions `yaml:"exclusions"`
	ExecutionMode string     `yaml:"execution_mode"` // "parallel" or "sequential"
}

func loadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML configuration: %w", err)
	}
	return &cfg, nil
}

/************* Exclusions *************/
func nowInLocation(tz string) time.Time {
	if tz == "" {
		return time.Now().UTC()
	}
	if l, err := time.LoadLocation(tz); err == nil {
		return time.Now().In(l)
	}
	return time.Now().UTC()
}

func parseDateOrPanic(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(fmt.Errorf("invalid date %q: %w", s, err))
	}
	return t
}

func sameDate(a, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func sameMonthDay(now, d time.Time) bool {
	return now.Month() == d.Month() && now.Day() == d.Day()
}

func inAnnualRange(now, from, to time.Time) bool {
	y := 2000
	n := time.Date(y, now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	f := time.Date(y, from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	t := time.Date(y, to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)
	if !t.Before(f) {
		return !n.Before(f) && !n.After(t)
	}
	return (!n.Before(f) && n.Before(time.Date(y, 12, 31, 0, 0, 0, 0, time.UTC).Add(24*time.Hour))) || (!n.After(t))
}

func inExclusions(cfg *Config, now time.Time) (bool, string) {
	for _, d := range cfg.Exclusions.Dates {
		dt := parseDateOrPanic(d.Date)
		if d.Annual && sameMonthDay(now, dt) {
			return true, fmt.Sprintf("excluded (annual day %s)", d.Date)
		}
		if !d.Annual && sameDate(now, dt) {
			return true, fmt.Sprintf("excluded (day %s)", d.Date)
		}
	}
	for _, r := range cfg.Exclusions.Ranges {
		f := parseDateOrPanic(r.From)
		t := parseDateOrPanic(r.To)
		if r.Annual && inAnnualRange(now, f, t) {
			return true, fmt.Sprintf("excluded (annual range %s..%s)", r.From, r.To)
		}
		if !r.Annual && !now.Before(f) && !now.After(t) {
			return true, fmt.Sprintf("excluded (range %s..%s)", r.From, r.To)
		}
	}
	return false, ""
}

func newLogger() *slog.Logger {
	level := slog.LevelInfo
	if strings.EqualFold(os.Getenv("LOG_LEVEL"), "debug") {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func main() {
	log := logging.NewSlogLogger(newLogger().With("app", "ce-vpc-power-schedule"))

	if len(os.Args) == 2 && (os.Args[1] == "powerOn" || os.Args[1] == "powerOff") {
		log.Info("Arguments", slog.String("action", os.Args[1]))
	} else {
		log.Error("missing or invalid argument (use: powerOn | powerOff)")
		return
	}

	action := os.Args[1]

	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "/app/config/schedule.yaml"
	}
	cfg, err := loadConfig(cfgPath)
	if err != nil {
		log.Error("Failed to load configuration", slog.Any("error", err))
		return
	}

	now := nowInLocation(cfg.Exclusions.Timezone)
	if excluded, why := inExclusions(cfg, now); excluded {
		log.Info("Skip execution, no action needed", slog.String("reason", why))
		return
	}

	apiKey := os.Getenv("IBM_APIKEY")
	region := os.Getenv("IBM_REGION")

	if apiKey == "" {
		log.Error("IBM_APIKEY environment variable is required")
		return
	}
	if region == "" {
		log.Error("IBM_REGION environment variable is required")
		return
	}

	vpcClient, err := vpc.NewVPCClient(&vpc.VPCClientOptions{
		APIKey: apiKey,
		Region: region,
		Log:    log,
	})
	if err != nil {
		log.Error("Failed to create VPC client", slog.Any("error", err))
		return
	}

	taggingClient, err := vpc.NewTaggingClient(apiKey, log)
	if err != nil {
		log.Error("Failed to create tagging client", slog.Any("error", err))
		return
	}

	rmClient, err := vpc.NewResourceManagerClient(apiKey, log)
	if err != nil {
		log.Error("Failed to create resource manager client", slog.Any("error", err))
		return
	}

	// Determine execution mode (default to parallel for backward compatibility)
	executionMode := cfg.ExecutionMode
	if executionMode == "" {
		executionMode = "parallel"
	}
	if executionMode != "parallel" && executionMode != "sequential" {
		log.Error("Invalid execution_mode", slog.String("mode", executionMode), slog.String("valid_values", "parallel, sequential"))
		return
	}

	log.Info("Execution mode", slog.String("mode", executionMode))

	for _, entity := range cfg.Entities {
		switch entity.Type {
		case instanceType:
			handleInstance(vpcClient, entity, action, executionMode, log)
		case tagType:
			handleTag(vpcClient, taggingClient, entity, action, executionMode, log)
		case placementGroupType:
			handlePlacementGroup(vpcClient, entity, action, executionMode, log)
		case resourceGroupType:
			handleResourceGroup(vpcClient, rmClient, entity, action, executionMode, log)
		case vpcType:
			handleVPC(vpcClient, entity, action, executionMode, log)
		default:
			log.Error("Unsupported entity type", slog.String("type", entity.Type))
		}
	}
}

func handleInstance(vpcClient *vpc.VPCClient, entity Entity, action string, executionMode string, log *logging.SlogLogger) {
	log.Info("Processing instance", slog.String("name", entity.Value))

	instances, err := vpcClient.ListInstancesByName(entity.Value)
	if err != nil {
		log.Error("Failed to list instances", slog.Any("error", err))
		return
	}

	if len(instances) == 0 {
		log.Warn("No instances found", slog.String("name", entity.Value))
		return
	}

	processInstances(vpcClient, instances, action, executionMode, log)
}

func handleTag(vpcClient *vpc.VPCClient, taggingClient *vpc.TaggingClient, entity Entity, action string, executionMode string, log *logging.SlogLogger) {
	log.Info("Processing instances with tag", slog.String("tag", entity.Value))

	// First, get CRNs of instances with the tag using Global Search
	crns, err := taggingClient.GetInstanceCRNsByTag(entity.Value)
	if err != nil {
		log.Error("Failed to get instance CRNs by tag", slog.Any("error", err))
		return
	}

	if len(crns) == 0 {
		log.Warn("No instances found with tag", slog.String("tag", entity.Value))
		return
	}

	// Then, get instance details from VPC using the CRNs
	instances, err := vpcClient.ListInstancesByTag(crns)
	if err != nil {
		log.Error("Failed to list instances by tag", slog.Any("error", err))
		return
	}

	if len(instances) == 0 {
		log.Warn("No valid instances found with tag", slog.String("tag", entity.Value))
		return
	}

	processInstances(vpcClient, instances, action, executionMode, log)
}

func handlePlacementGroup(vpcClient *vpc.VPCClient, entity Entity, action string, executionMode string, log *logging.SlogLogger) {
	log.Info("Processing instances in placement group", slog.String("placementGroup", entity.Value))

	instances, err := vpcClient.ListInstancesByPlacementGroup(entity.Value)
	if err != nil {
		log.Error("Failed to list instances by placement group", slog.Any("error", err))
		return
	}

	if len(instances) == 0 {
		log.Warn("No instances found in placement group", slog.String("placementGroup", entity.Value))
		return
	}

	processInstances(vpcClient, instances, action, executionMode, log)
}

func handleResourceGroup(vpcClient *vpc.VPCClient, rmClient *vpc.ResourceManagerClient, entity Entity, action string, executionMode string, log *logging.SlogLogger) {
	log.Info("Processing instances in resource group", slog.String("resourceGroup", entity.Value))

	instances, err := vpcClient.ListInstancesByResourceGroup(entity.Value, rmClient)
	if err != nil {
		log.Error("Failed to list instances by resource group", slog.Any("error", err))
		return
	}

	if len(instances) == 0 {
		log.Warn("No instances found in resource group", slog.String("resourceGroup", entity.Value))
		return
	}

	processInstances(vpcClient, instances, action, executionMode, log)
}

func handleVPC(vpcClient *vpc.VPCClient, entity Entity, action string, executionMode string, log *logging.SlogLogger) {
	log.Info("Processing instances in VPC", slog.String("vpc", entity.Value))

	instances, err := vpcClient.ListInstancesByVPC(entity.Value)
	if err != nil {
		log.Error("Failed to list instances by VPC", slog.Any("error", err))
		return
	}

	if len(instances) == 0 {
		log.Warn("No instances found in VPC", slog.String("vpc", entity.Value))
		return
	}

	processInstances(vpcClient, instances, action, executionMode, log)
}

// processInstances processes multiple instances based on execution mode
func processInstances(vpcClient *vpc.VPCClient, instances []*vpc.Instance, action string, executionMode string, log *logging.SlogLogger) {
	startTime := time.Now()

	if executionMode == "sequential" {
		log.Info("Starting sequential processing",
			slog.Int("instanceCount", len(instances)),
			slog.String("action", action))

		for _, inst := range instances {
			processInstance(vpcClient, inst, action, log)
		}

		duration := time.Since(startTime)
		log.Info("Sequential processing completed",
			slog.Int("instanceCount", len(instances)),
			slog.String("action", action),
			slog.Duration("duration", duration),
			slog.String("durationFormatted", duration.Round(time.Millisecond).String()))
	} else {
		// Parallel execution
		var wg sync.WaitGroup

		log.Info("Starting parallel processing",
			slog.Int("instanceCount", len(instances)),
			slog.String("action", action))

		for _, inst := range instances {
			wg.Add(1)
			go func(instance *vpc.Instance) {
				defer wg.Done()
				processInstance(vpcClient, instance, action, log)
			}(inst)
		}

		wg.Wait()
		duration := time.Since(startTime)
		log.Info("Parallel processing completed",
			slog.Int("instanceCount", len(instances)),
			slog.String("action", action),
			slog.Duration("duration", duration),
			slog.String("durationFormatted", duration.Round(time.Millisecond).String()))
	}
}

func processInstance(vpcClient *vpc.VPCClient, inst *vpc.Instance, action string, log *logging.SlogLogger) {
	startTime := time.Now()
	log.Info("Processing instance",
		slog.String("id", inst.ID),
		slog.String("name", inst.Name),
		slog.String("status", inst.Status))

	switch action {
	case "powerOff":
		if strings.EqualFold(inst.Status, "running") {
			err := vpcClient.StopInstance(inst.ID)
			if err != nil {
				log.Error("Failed to stop instance",
					slog.String("id", inst.ID),
					slog.String("name", inst.Name),
					slog.Any("error", err))
				return
			}
			duration := time.Since(startTime)
			log.Info("Instance stopped successfully",
				slog.String("id", inst.ID),
				slog.String("name", inst.Name),
				slog.Duration("duration", duration),
				slog.String("durationFormatted", duration.Round(time.Millisecond).String()))
		} else {
			log.Warn("Instance is not running, no action needed",
				slog.String("id", inst.ID),
				slog.String("name", inst.Name),
				slog.String("status", inst.Status))
		}
	case "powerOn":
		if strings.EqualFold(inst.Status, "stopped") {
			err := vpcClient.StartInstance(inst.ID)
			if err != nil {
				log.Error("Failed to start instance",
					slog.String("id", inst.ID),
					slog.String("name", inst.Name),
					slog.Any("error", err))
				return
			}
			duration := time.Since(startTime)
			log.Info("Instance started successfully",
				slog.String("id", inst.ID),
				slog.String("name", inst.Name),
				slog.Duration("duration", duration),
				slog.String("durationFormatted", duration.Round(time.Millisecond).String()))
		} else {
			log.Warn("Instance is not stopped, no action needed",
				slog.String("id", inst.ID),
				slog.String("name", inst.Name),
				slog.String("status", inst.Status))
		}
	default:
		log.Error("Unsupported action", slog.String("action", action))
	}
}
